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
	connMap := make(map[string]*conn)
	queueMap := make(map[string]*Queue)

	// 为每个唯一的 URL 创建 conn
	makeConn(connMap, configs.Config.AMQP091Queue.OnOpen)
	makeConn(connMap, configs.Config.AMQP091Queue.OnMessage)
	makeConn(connMap, configs.Config.AMQP091Queue.OnClose)

	// 创建队列
	Manager[int(netsvrProtocol.Event_OnOpen)] = makeQueue(connMap, queueMap, configs.Config.AMQP091Queue.OnOpen, 256)
	Manager[int(netsvrProtocol.Event_OnMessage)] = makeQueue(connMap, queueMap, configs.Config.AMQP091Queue.OnMessage, 256)
	Manager[int(netsvrProtocol.Event_OnClose)] = makeQueue(connMap, queueMap, configs.Config.AMQP091Queue.OnClose, 256)

	// 打印启动日志
	for _, q := range queueMap {
		if q == nil {
			continue
		}
		log.Logger.Info().
			Int("pid", os.Getpid()).
			Str("address", q.channel.conn.address).
			Str("exchange", q.channel.exchange).
			Str("queue", q.channel.queue).
			Str("routingKey", q.channel.routingKey).
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
			log.Logger.Info().
				Int("pid", os.Getpid()).
				Str("address", q.channel.conn.address).
				Str("exchange", q.channel.exchange).
				Str("queue", q.channel.queue).
				Str("routingKey", q.channel.routingKey).
				Msg("AMQP091 Queue shutdown")
		}(q)
	}
	wg.Wait()
}

// makeConn 创建一个连接管理器
func makeConn(connMap map[string]*conn, queueConfig configs.AMQP091Queue) {
	if queueConfig.Address == "" || queueConfig.Exchange == "" {
		// 没有配置
		return
	}
	// 使用 Address、Username 作为 key
	connId := fmt.Sprintf("Address%sUsername%s", queueConfig.Address, queueConfig.Username)
	if connMap[connId] != nil {
		// 已经初始化过了
		return
	}
	// URL 编码用户名和密码（处理特殊字符）
	username := url.QueryEscape(queueConfig.Username)
	password := url.QueryEscape(queueConfig.Password)
	urlStr := fmt.Sprintf("amqp://%s:%s@%s%s", username, password, queueConfig.Address, *queueConfig.VHost)
	if conn := newConn(urlStr, queueConfig.Address); conn != nil {
		connMap[connId] = conn
	}
}

// makeQueue 创建一个队列
func makeQueue(connMap map[string]*conn, queueMap map[string]*Queue, queueConfig configs.AMQP091Queue, dequeueSize int) *Queue {
	if queueConfig.Address == "" || queueConfig.Exchange == "" {
		// 没有配置
		return nil
	}

	queueId := fmt.Sprintf("Address%sExchange%sQueue%sRoutingKey%s",
		queueConfig.Address,
		queueConfig.Exchange,
		*queueConfig.Queue,
		*queueConfig.RoutingKey,
	)
	if queueMap[queueId] != nil {
		// 已经初始化过了，说明同一个队列可以处理多个Event，直接返回
		return queueMap[queueId]
	}

	// 使用 Address、Username  作为 key 查找 conn
	connId := fmt.Sprintf("Address%sUsername%s", queueConfig.Address, queueConfig.Username)
	conn := connMap[connId]
	if conn == nil {
		return nil
	}
	amqpChannel := newChannel(conn, queueConfig)
	if amqpChannel == nil {
		return nil
	}
	q := newQueue(amqpChannel, *queueConfig.Durable, dequeueSize)
	queueMap[queueId] = q
	return q
}
