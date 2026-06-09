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

// Package amqp091 AMQP队列
package amqp091

import (
	"fmt"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"net/url"
	"netsvr/configs"
	"netsvr/internal/log"
	"os"
	"runtime"
	"sync"
)

// 数组大小基于协议中最大的 Event 枚举值
const managerLen = netsvrProtocol.Event_OnMessage + 1

type manager [managerLen]*Queue

func (r manager) Get(event netsvrProtocol.Event) *Queue {
	return r[event]
}

// Manager 管理所有的AMQP队列
var Manager manager

func init() {
	// 验证协议中的 Event 枚举值是否超出数组范围
	maxUsedEvent := 0
	for _, v := range netsvrProtocol.Event_value {
		maxUsedEvent = max(maxUsedEvent, int(v))
	}
	if maxUsedEvent >= int(managerLen) {
		log.Logger.Error().Msgf("Event enum value %d exceeds manager array size %d",
			maxUsedEvent, managerLen)
		panic("too many netsvrProtocol.Event")
	}
	Manager = manager{}
}

// Start 启动AMQP队列
func Start() {
	connChannelPoolMap := make(map[string]*channelPool)
	topologyMap := make(map[string]bool)
	queueMap := make(map[string]*Queue)
	dequeueSize := 256
	// 为每个唯一的 URL 创建 connPool与channelPool
	makeConnChannel(connChannelPoolMap, configs.Config.AMQP091Queue.OnOpen, dequeueSize)
	makeConnChannel(connChannelPoolMap, configs.Config.AMQP091Queue.OnMessage, dequeueSize)
	makeConnChannel(connChannelPoolMap, configs.Config.AMQP091Queue.OnClose, dequeueSize)

	// 创建拓扑结构
	makeTopology(connChannelPoolMap, topologyMap, configs.Config.AMQP091Queue.OnOpen)
	makeTopology(connChannelPoolMap, topologyMap, configs.Config.AMQP091Queue.OnMessage)
	makeTopology(connChannelPoolMap, topologyMap, configs.Config.AMQP091Queue.OnClose)

	// 创建队列
	Manager[int(netsvrProtocol.Event_OnOpen)] = makeQueue(connChannelPoolMap, queueMap, configs.Config.AMQP091Queue.OnOpen, dequeueSize)
	Manager[int(netsvrProtocol.Event_OnMessage)] = makeQueue(connChannelPoolMap, queueMap, configs.Config.AMQP091Queue.OnMessage, dequeueSize)
	Manager[int(netsvrProtocol.Event_OnClose)] = makeQueue(connChannelPoolMap, queueMap, configs.Config.AMQP091Queue.OnClose, dequeueSize)

	// 打印启动日志
	for _, q := range queueMap {
		if q == nil {
			continue
		}
		connPoolSize, channelPoolSize := getPoolSize(runtime.NumCPU())
		log.Logger.Info().
			Int("pid", os.Getpid()).
			Int("connPoolSize", connPoolSize).
			Int("channelPoolSize", channelPoolSize).
			Str("address", q.channelPool.connPool.address).
			Str("exchange", q.exchange).
			Str("routingKey", q.routingKey).
			Msg("AMQP091 Queue start")
	}
}

// Shutdown 停止AMQP队列
func Shutdown() {
	wg := &sync.WaitGroup{}
	for _, q := range Manager {
		if q == nil {
			continue
		}
		wg.Add(1)
		go func(queue *Queue) {
			defer func() {
				wg.Done()
			}()
			if !queue.close() {
				return
			}
			connPoolSize, channelPoolSize := getPoolSize(runtime.NumCPU())
			log.Logger.Info().
				Int("pid", os.Getpid()).
				Int("connPoolSize", connPoolSize).
				Int("channelPoolSize", channelPoolSize).
				Str("address", q.channelPool.connPool.address).
				Str("exchange", q.exchange).
				Str("routingKey", q.routingKey).
				Msg("AMQP091 Queue shutdown")
		}(q)
	}
	wg.Wait()
}

// makeConnChannel 创建一个连接管理器
func makeConnChannel(connChannelPoolMap map[string]*channelPool, queueConfig configs.AMQP091Queue, dequeueSize int) {
	if queueConfig.Address == "" || queueConfig.Exchange == "" {
		// 没有配置
		return
	}
	// 使用 Address、Username 作为 key
	connChannelPoolId := fmt.Sprintf("Address%sUsername%s", queueConfig.Address, queueConfig.Username)
	if connChannelPoolMap[connChannelPoolId] != nil {
		// 已经初始化过了
		return
	}
	// URL 编码用户名和密码（处理特殊字符）
	username := url.QueryEscape(queueConfig.Username)
	password := url.QueryEscape(queueConfig.Password)
	urlStr := fmt.Sprintf("amqp://%s:%s@%s%s", username, password, queueConfig.Address, *queueConfig.VHost)
	connPoolSize, channelPoolSize := getPoolSize(runtime.NumCPU())
	if conn := newConnPool(urlStr, queueConfig.Address, connPoolSize); conn != nil {
		pool := newChannelPool(conn, channelPoolSize, dequeueSize)
		if pool == nil {
			return
		}
		connChannelPoolMap[connChannelPoolId] = pool
	}
}

// makeTopology 创建一个拓扑结构
func makeTopology(connPoolMap map[string]*channelPool, topologyMap map[string]bool, queueConfig configs.AMQP091Queue) {
	if queueConfig.Address == "" || queueConfig.Exchange == "" {
		// 没有配置
		return
	}

	topologyId := fmt.Sprintf("Address%sExchange%sQueue%sRoutingKey%s",
		queueConfig.Address,
		queueConfig.Exchange,
		*queueConfig.Queue,
		*queueConfig.RoutingKey,
	)
	if topologyMap[topologyId] {
		return
	}

	// 使用 Address、Username  作为 key 查找 connChannelPool
	connChannelPoolId := fmt.Sprintf("Address%sUsername%s", queueConfig.Address, queueConfig.Username)
	chPool := connPoolMap[connChannelPoolId]
	if chPool == nil {
		return
	}
	amqpChannel := chPool.getAmqpChannel()
	if amqpChannel == nil {
		return
	}
	defer chPool.release(amqpChannel)
	// 声明 Exchange
	if err := amqpChannel.channel.ExchangeDeclare(
		queueConfig.Exchange,
		queueConfig.ExchangeType,
		true,                    // durable: 持久化交换机
		*queueConfig.AutoDelete, // autoDelete: 自动删除
		false,                   // internal: 内部交换机
		false,                   // noWait: 不等待响应
		nil,                     // arguments
	); err != nil {
		log.Logger.Error().Err(err).
			Str("address", chPool.connPool.address).
			Str("exchange", queueConfig.Exchange).
			Str("queue", *queueConfig.Queue).
			Str("routingKey", *queueConfig.RoutingKey).
			Msg("AMQP091 declare exchange failed")
		return
	}

	// 如果配置了 Queue 名称，则声明队列并绑定
	if *queueConfig.Queue != "" {
		_, err := amqpChannel.channel.QueueDeclare(
			*queueConfig.Queue,
			true,                    // durable: 持久化队列
			*queueConfig.AutoDelete, // autoDelete: 自动删除
			false,                   // exclusive: 非独占
			false,                   // noWait: 不等待响应
			nil,                     // arguments
		)
		if err != nil {
			log.Logger.Error().Err(err).
				Str("address", chPool.connPool.address).
				Str("exchange", queueConfig.Exchange).
				Str("queue", *queueConfig.Queue).
				Str("routingKey", *queueConfig.RoutingKey).
				Msg("AMQP091 declare queue failed")
			return
		}

		if err := amqpChannel.channel.QueueBind(
			*queueConfig.Queue,
			*queueConfig.RoutingKey,
			queueConfig.Exchange,
			false, // noWait: 不等待响应
			nil,   // arguments
		); err != nil {
			log.Logger.Error().Err(err).
				Str("address", chPool.connPool.address).
				Str("exchange", queueConfig.Exchange).
				Str("queue", *queueConfig.Queue).
				Str("routingKey", *queueConfig.RoutingKey).
				Msg("AMQP091 bind queue failed")
			return
		}
	}

	log.Logger.Info().
		Str("address", chPool.connPool.address).
		Str("exchange", queueConfig.Exchange).
		Str("queue", *queueConfig.Queue).
		Str("routingKey", *queueConfig.RoutingKey).
		Msg("AMQP091 topology declared")

	topologyMap[topologyId] = true
}

// makeQueue 创建一个队列
func makeQueue(connChannelPoolMap map[string]*channelPool, queueMap map[string]*Queue, queueConfig configs.AMQP091Queue, dequeueSize int) *Queue {
	if queueConfig.Address == "" || queueConfig.Exchange == "" {
		// 没有配置
		return nil
	}

	queueId := fmt.Sprintf("Address%sExchange%sRoutingKey%s",
		queueConfig.Address,
		queueConfig.Exchange,
		*queueConfig.RoutingKey,
	)
	if queueMap[queueId] != nil {
		// 已经初始化过了，说明同一个队列可以处理多个Event，直接返回
		return queueMap[queueId]
	}

	// 使用 Address、Username  作为 key 查找 connChannelPool
	connChannelPoolId := fmt.Sprintf("Address%sUsername%s", queueConfig.Address, queueConfig.Username)
	chPool := connChannelPoolMap[connChannelPoolId]
	if chPool == nil {
		return nil
	}
	q := newQueue(chPool, queueConfig, dequeueSize)
	queueMap[queueId] = q
	return q
}

// 根据cpu核数计算mq的tcp连接总数以及channel总数
func getPoolSize(cpu int) (connPoolSize int, channelPoolSize int) {
	var offset int
	if cpu <= 8 {
		offset = 1
	} else if cpu <= 16 {
		offset = 2
	} else if cpu <= 32 {
		offset = 3
	} else {
		offset = 5
	}
	connPoolSize = max(cpu/4, 1) + offset
	if cpu < 4 {
		connPoolSize = 1
	}
	channelPoolSize = connPoolSize * 12
	return
}
