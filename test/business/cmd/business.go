/**
* Copyright 2023 buexplain@qq.com
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

// 对接网关支持的各种操作接口
package main

import (
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	_ "github.com/buexplain/netsvr-business-go/v2"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"netsvr/pkg/quit"
	"netsvr/test/business/assets"
	"netsvr/test/business/configs"
	"netsvr/test/business/internal/cmd"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/mainSocketManager"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/pkg/protocol"
	"os"
	"strings"
	"time"
	"unsafe"
)

func main() {
	//启动html客户端的服务器
	go clientServer()
	//启动redis队列消费者
	go redisQueueConsumer()
	//启动amqp091队列消费者
	go amqp091QueueConsumer()
	//启动worker连接
	if configs.Config.WorkerListenAddress != "" {
		if mainSocketManager.MainSocketManager.Start() == false {
			log.Logger.Error().Msg("注册到worker服务器失败")
			os.Exit(1)
		} else {
			log.Logger.Debug().Msg("注册到worker服务器成功")
		}
	}
	//处理关闭信号
	quit.Wg.Add(1)
	go func() {
		defer func() {
			_ = recover()
			quit.Wg.Done()
		}()
		<-quit.Ctx.Done()
		mainSocketManager.MainSocketManager.Close()
		netBus.NetBus.Close()
	}()
	//开始关闭进程
	select {
	case <-quit.ClosedCh:
		//及时打印关闭进程的日志，避免使用者认为进程无反应，直接强杀进程
		log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("开始关闭business进程")
		//通知所有协程开始退出
		quit.Cancel()
		//等待协程退出
		quit.Wg.Wait()
		log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("关闭business进程成功")
		os.Exit(0)
	}
}

// 输出html客户端
// 提供websocket连接发消息、打开、关闭的回调api
func clientServer() {
	if configs.Config.ClientListenAddress == "" {
		return
	}
	checkIsOpen := func(addr string) bool {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			_ = c.Close()
			return true
		}
		var e *net.OpError
		if errors.As(err, &e) && (strings.Contains(e.Err.Error(), "No connection") || strings.Contains(e.Err.Error(), "connection refused")) {
			return false
		}
		return true
	}
	if checkIsOpen(configs.Config.ClientListenAddress) {
		log.Logger.Info().Msg("地址已被占用: " + configs.Config.ClientListenAddress)
		return
	}
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		t, err := template.New("").Delims("{!", "!}").Parse(assets.GetClientHtml())
		if err != nil {
			log.Logger.Error().Err(err).Msg("模板解析失败")
			return
		}
		data := map[string]interface{}{}
		//注入连接地址
		connUrl := configs.Config.CustomerWsAddress
		data["conn"] = connUrl
		//把所有的命令注入到客户端
		for c, name := range protocol.CmdName {
			data[name] = int(c)
		}
		data["heartbeatMessage"] = string(configs.Config.CustomerHeartbeatMessage)
		err = t.Execute(writer, data)
		if err != nil {
			log.Logger.Error().Msgf("模板输出失败：%s", err)
			return
		}
	})
	//监听onopen
	http.HandleFunc("/onopen", func(writer http.ResponseWriter, request *http.Request) {
		protobuf, err := io.ReadAll(request.Body)
		if err != nil {
			log.Logger.Error().Msgf("读取回调的netsvrProtocol.ConnOpen失败：%s", err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		cp := &netsvrProtocol.ConnOpen{}
		if err := proto.Unmarshal(protobuf, cp); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			log.Logger.Error().Msgf("解析回调的netsvrProtocol.ConnOpen失败：%s", err)
			return
		}
		cpResp := &netsvrProtocol.ConnOpenResp{
			Allow: true,
		}
		responseData, err := proto.Marshal(cpResp)
		if err != nil {
			log.Logger.Error().Msgf("序列化回调的netsvrProtocol.ConnOpenResp失败：%s", err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		writer.Header().Set("Content-Type", "application/x-protobuf")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(responseData)
		if configs.Config.Service == "callback" {
			go cmd.EventHandler.OnOpen(cp)
		}
	})
	//监听onmessage
	http.HandleFunc("/onmessage", func(writer http.ResponseWriter, request *http.Request) {
		protobuf, err := io.ReadAll(request.Body)
		if err != nil {
			log.Logger.Error().Msgf("读取回调的netsvrProtocol.Transfer失败：%s", err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		tf := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(protobuf, tf); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			log.Logger.Error().Msgf("解析回调的netsvrProtocol.Transfer失败：%s", err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
		if configs.Config.Service == "callback" {
			go cmd.EventHandler.OnMessage(tf)
		}
	})
	//监听onclose
	http.HandleFunc("/onclose", func(writer http.ResponseWriter, request *http.Request) {
		protobuf, err := io.ReadAll(request.Body)
		if err != nil {
			log.Logger.Error().Msgf("读取回调的netsvrProtocol.ConnClose失败：%s", err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		cc := &netsvrProtocol.ConnClose{}
		if err := proto.Unmarshal(protobuf, cc); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			log.Logger.Error().Msgf("解析回调的netsvrProtocol.ConnClose失败：%s", err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
		if configs.Config.Service == "callback" {
			go cmd.EventHandler.OnClose(cc)
		}
	})
	log.Logger.Info().Msg("当前提供服务的是：" + configs.Config.Service)
	log.Logger.Info().Msg("点击访问客户端：http" + ":" + "//" + configs.Config.ClientListenAddress + "/")
	_ = http.ListenAndServe(configs.Config.ClientListenAddress, nil)
}

// redis队列消费者
func redisQueueConsumer() {
	// 定义消费者函数
	consumeList := func(queueConfig configs.RedisQueue, handler func([]byte, string)) {
		if queueConfig.Address == "" || queueConfig.Key == "" {
			return
		}

		client := redis.NewClient(&redis.Options{
			Addr:     queueConfig.Address,
			Password: *queueConfig.Password,
			DB:       *queueConfig.DB,
		})
		defer func(client *redis.Client) {
			_ = client.Close()
		}(client)

		log.Logger.Info().Str("key", queueConfig.Key).Str("keyType", queueConfig.KeyType).Msg("Redis队列消费者启动")

		// Stream 消费者需要记录上次消费的最大消息 ID，避免反复消费
		// 初始为 "0" 表示从头消费所有积压消息，后续用实际消息 ID 继续读取
		lastStreamID := "0"
		pipe := client.Pipeline()
		for {
			select {
			case <-quit.Ctx.Done():
				log.Logger.Info().Str("key", queueConfig.Key).Str("keyType", queueConfig.KeyType).Msg("Redis队列消费者退出")
				return
			default:
			}
			if queueConfig.KeyType == "list" {
				for i := 0; i < 10; i++ {
					// 从列表右侧弹出消息
					pipe.RPop(quit.Ctx, queueConfig.Key)
				}
				cmderList, err := pipe.Exec(quit.Ctx)
				if err != nil && !errors.Is(err, redis.Nil) {
					log.Logger.Error().Err(err).Str("key", queueConfig.Key).Str("keyType", queueConfig.KeyType).Msg("RPop失败")
					continue
				}
				for _, cmder := range cmderList {
					if cmder.Err() != nil {
						if !errors.Is(err, redis.Nil) {
							log.Logger.Error().Err(cmder.Err()).Str("key", queueConfig.Key).Str("keyType", queueConfig.KeyType).Msg("RPop失败")
						}
						continue
					}
					stringCmd := cmder.(*redis.StringCmd)
					if str, err := stringCmd.Result(); err != nil {
						if !errors.Is(err, redis.Nil) {
							log.Logger.Error().Err(err).Str("key", queueConfig.Key).Str("keyType", queueConfig.KeyType).Msg("RPop失败")
						}
						continue
					} else {
						handler(unsafe.Slice(unsafe.StringData(str), len(str)), "redis")
					}
				}
			} else if queueConfig.KeyType == "stream" {
				// 从Stream读取消息，起始 ID 为上次消费的最大消息 ID，避免反复消费
				streams, err := client.XRead(quit.Ctx, &redis.XReadArgs{
					Streams: []string{queueConfig.Key, lastStreamID},
					Count:   10,
					Block:   time.Second * 3,
				}).Result()
				if err != nil {
					if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, redis.Nil) {
						log.Logger.Error().Err(err).Str("key", queueConfig.Key).Str("keyType", queueConfig.KeyType).Msg("XRead失败")
					}
					continue
				}
				for _, stream := range streams {
					for _, msg := range stream.Messages {
						// Stream的Values是map[string]interface{}，网关将数据存储在"data"字段中
						if data, ok := msg.Values["data"].(string); ok {
							handler(unsafe.Slice(unsafe.StringData(data), len(data)), "redis")
						}
						// 删除已处理的消息，防止反复消费
						pipe.XDel(quit.Ctx, queueConfig.Key, msg.ID)
						// 更新 lastStreamID 为当前批次最大消息 ID
						lastStreamID = msg.ID
					}
				}
				_, err = pipe.Exec(quit.Ctx)
				if err != nil {
					pipe.Discard()
					log.Logger.Error().Err(err).Str("key", queueConfig.Key).Str("keyType", queueConfig.KeyType).Msg("XDel失败")
				}
			}
		}
	}
	queueConfigs := make([]configs.RedisQueue, 0, 3)
	queueConfigs = append(queueConfigs, configs.Config.RedisQueue.OnOpen)
	queueConfigs = append(queueConfigs, configs.Config.RedisQueue.OnMessage)
	queueConfigs = append(queueConfigs, configs.Config.RedisQueue.OnClose)
	queueIdMap := map[string]bool{}
	for _, queueConfig := range queueConfigs {
		queueId := fmt.Sprintf("Address%sDB%dKey%sKeyType%s",
			queueConfig.Address,
			queueConfig.DB,
			queueConfig.Key,
			queueConfig.KeyType,
		)
		if queueIdMap[queueId] {
			continue
		}
		queueIdMap[queueId] = true
		go consumeList(queueConfig, dispatchHandler)
	}
}

// AMQP队列的消费者
func amqp091QueueConsumer() {
	if configs.Config.AMQP091Queue.Address == "" || configs.Config.AMQP091Queue.Exchange == "" {
		return
	}
	// 定义AMQP消费者函数
	consumeAMQP := func(queueConfig configs.AMQP091Queue, handler func([]byte, string)) {
		if queueConfig.Address == "" || queueConfig.Exchange == "" || *queueConfig.Queue == "" {
			return
		}

		// URL 编码用户名和密码（处理特殊字符）
		username := url.QueryEscape(queueConfig.Username)
		password := url.QueryEscape(queueConfig.Password)
		urlStr := fmt.Sprintf("amqp://%s:%s@%s%s", username, password, queueConfig.Address, *queueConfig.VHost)

		log.Logger.Info().
			Str("address", queueConfig.Address).
			Str("exchange", queueConfig.Exchange).
			Str("queue", *queueConfig.Queue).
			Str("routingKey", *queueConfig.RoutingKey).
			Msg("AMQP091队列消费者启动")

		for {
			select {
			case <-quit.Ctx.Done():
				log.Logger.Info().
					Str("address", queueConfig.Address).
					Str("exchange", queueConfig.Exchange).
					Str("queue", *queueConfig.Queue).
					Msg("AMQP091队列消费者退出")
				return
			default:
			}

			// 建立连接
			conn, err := amqp.Dial(urlStr)
			if err != nil {
				log.Logger.Error().Err(err).
					Str("address", queueConfig.Address).
					Str("exchange", queueConfig.Exchange).
					Str("queue", *queueConfig.Queue).
					Msg("AMQP091连接失败，3秒后重试")
				time.Sleep(time.Second * 3)
				continue
			}

			// 创建通道
			ch, err := conn.Channel()
			if err != nil {
				_ = conn.Close()
				log.Logger.Error().Err(err).
					Str("address", queueConfig.Address).
					Str("exchange", queueConfig.Exchange).
					Str("queue", *queueConfig.Queue).
					Msg("AMQP091创建通道失败，3秒后重试")
				time.Sleep(time.Second * 3)
				continue
			}

			// 声明队列（确保队列存在）
			_, err = ch.QueueDeclare(
				*queueConfig.Queue,
				true,  // durable: 持久化队列
				false, // autoDelete: 不自动删除
				false, // exclusive: 非独占
				false, // noWait: 不等待响应
				nil,   // arguments
			)
			if err != nil {
				_ = ch.Close()
				_ = conn.Close()
				log.Logger.Error().Err(err).
					Str("address", queueConfig.Address).
					Str("exchange", queueConfig.Exchange).
					Str("queue", *queueConfig.Queue).
					Msg("AMQP091声明队列失败，3秒后重试")
				time.Sleep(time.Second * 3)
				continue
			}

			// 设置QoS，每次只预取一条消息
			err = ch.Qos(
				1,     // prefetch count
				0,     // prefetch size
				false, // global
			)
			if err != nil {
				_ = ch.Close()
				_ = conn.Close()
				log.Logger.Error().Err(err).
					Str("address", queueConfig.Address).
					Str("exchange", queueConfig.Exchange).
					Str("queue", *queueConfig.Queue).
					Msg("AMQP091设置QoS失败，3秒后重试")
				time.Sleep(time.Second * 3)
				continue
			}

			// 消费消息
			msgs, err := ch.Consume(
				*queueConfig.Queue,
				"",    // consumer: 空字符串表示自动生成消费者标签
				false, // autoAck: 手动确认
				false, // exclusive: 非独占
				false, // noLocal: 允许接收发布者自己的消息
				false, // noWait: 不等待响应
				nil,   // args
			)
			if err != nil {
				_ = ch.Close()
				_ = conn.Close()
				log.Logger.Error().Err(err).
					Str("address", queueConfig.Address).
					Str("exchange", queueConfig.Exchange).
					Str("queue", *queueConfig.Queue).
					Msg("AMQP091开始消费失败，3秒后重试")
				time.Sleep(time.Second * 3)
				continue
			}

			log.Logger.Info().
				Str("address", queueConfig.Address).
				Str("exchange", queueConfig.Exchange).
				Str("queue", *queueConfig.Queue).
				Msg("AMQP091开始消费消息")

			// 监听连接关闭通知
			notifyClose := conn.NotifyClose(make(chan *amqp.Error))

			// 消费消息循环
			consumeLoop := true
			for consumeLoop {
				select {
				case <-quit.Ctx.Done():
					consumeLoop = false
				case err := <-notifyClose:
					log.Logger.Error().Err(err).
						Str("address", queueConfig.Address).
						Str("exchange", queueConfig.Exchange).
						Str("queue", *queueConfig.Queue).
						Msg("AMQP091连接关闭，将重新连接")
					consumeLoop = false
				case msg, ok := <-msgs:
					if !ok {
						log.Logger.Error().
							Str("address", queueConfig.Address).
							Str("exchange", queueConfig.Exchange).
							Str("queue", *queueConfig.Queue).
							Msg("AMQP091消息通道关闭，将重新连接")
						consumeLoop = false
						break
					}
					// 处理消息
					handler(msg.Body, "amqp091")
					// 手动确认消息
					if err := msg.Ack(false); err != nil {
						log.Logger.Error().Err(err).
							Str("address", queueConfig.Address).
							Str("exchange", queueConfig.Exchange).
							Str("queue", *queueConfig.Queue).
							Msg("AMQP091确认消息失败")
					}
				}
			}

			// 关闭连接和通道
			_ = ch.Close()
			_ = conn.Close()

			// 如果不是主动退出，则等待后重连
			select {
			case <-quit.Ctx.Done():
				return
			default:
				log.Logger.Info().
					Str("address", queueConfig.Address).
					Str("exchange", queueConfig.Exchange).
					Str("queue", *queueConfig.Queue).
					Msg("AMQP091将在3秒后重新连接")
				time.Sleep(time.Second * 3)
			}
		}
	}
	queueConfigs := make([]configs.AMQP091Queue, 0, 3)
	queueConfigs = append(queueConfigs, configs.Config.AMQP091Queue.OnOpen)
	queueConfigs = append(queueConfigs, configs.Config.AMQP091Queue.OnMessage)
	queueConfigs = append(queueConfigs, configs.Config.AMQP091Queue.OnClose)
	queueIdMap := map[string]bool{}
	for _, queueConfig := range queueConfigs {
		queueId := fmt.Sprintf("Address%sExchange%sQueue%sRoutingKey%s",
			queueConfig.Address,
			queueConfig.Exchange,
			*queueConfig.Queue,
			*queueConfig.RoutingKey,
		)
		if queueIdMap[queueId] {
			continue
		}
		queueIdMap[queueId] = true
		go consumeAMQP(queueConfig, dispatchHandler)
	}
}

// 通用分派函数：从 4 字节 cmd 中解析命令类型，再根据 cmd 反序列化对应的 proto 对象
// 网关写入 队列 的数据格式为 [4字节cmd][proto body]
// 同一个队列可能包含多种事件类型（如 OnOpen + OnClose 共用一个 Redis key），必须先解析 cmd 再分派
func dispatchHandler(data []byte, service string) {
	// 只有队列服务才处理数据
	if configs.Config.Service != service {
		return
	}
	if len(data) < 4 {
		log.Logger.Error().Int("dataLen", len(data)).Msg("数据长度不足4字节，无法解析cmd")
		return
	}
	currentCmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(data[0:4]))
	body := data[4:]
	switch currentCmd {
	case netsvrProtocol.Cmd_ConnOpen:
		co := &netsvrProtocol.ConnOpen{}
		if err := proto.Unmarshal(body, co); err != nil {
			log.Logger.Error().Err(err).Msg("解析ConnOpen失败")
			return
		}
		go cmd.EventHandler.OnOpen(co)
	case netsvrProtocol.Cmd_Transfer:
		tf := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(body, tf); err != nil {
			log.Logger.Error().Err(err).Msg("解析Transfer失败")
			return
		}
		go cmd.EventHandler.OnMessage(tf)
	case netsvrProtocol.Cmd_ConnClose:
		cc := &netsvrProtocol.ConnClose{}
		if err := proto.Unmarshal(body, cc); err != nil {
			log.Logger.Error().Err(err).Msg("解析ConnClose失败")
			return
		}
		go cmd.EventHandler.OnClose(cc)
	default:
		log.Logger.Error().Str("cmd", currentCmd.String()).Msg("未知的cmd类型")
	}
}
