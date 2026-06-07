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
	queueMap := make(map[string]*Queue)
	dequeueSize := 256
	// 为每个唯一的 URL 创建 connPool与channelPool
	makeConnChannel(connChannelPoolMap, configs.Config.AMQP091Queue.OnOpen, dequeueSize)
	makeConnChannel(connChannelPoolMap, configs.Config.AMQP091Queue.OnMessage, dequeueSize)
	makeConnChannel(connChannelPoolMap, configs.Config.AMQP091Queue.OnClose, dequeueSize)

	// 创建队列
	Manager[int(netsvrProtocol.Event_OnOpen)] = makeQueue(connChannelPoolMap, queueMap, configs.Config.AMQP091Queue.OnOpen, 256)
	Manager[int(netsvrProtocol.Event_OnMessage)] = makeQueue(connChannelPoolMap, queueMap, configs.Config.AMQP091Queue.OnMessage, 256)
	Manager[int(netsvrProtocol.Event_OnClose)] = makeQueue(connChannelPoolMap, queueMap, configs.Config.AMQP091Queue.OnClose, 256)

	// 打印启动日志
	for _, q := range queueMap {
		if q == nil {
			continue
		}
		log.Logger.Info().
			Int("pid", os.Getpid()).
			Str("address", q.channelPool.connPool.address).
			Str("exchange", q.channelPool.exchange).
			Str("queue", q.channelPool.queue).
			Str("routingKey", q.channelPool.routingKey).
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
				Str("address", q.channelPool.connPool.address).
				Str("exchange", q.channelPool.exchange).
				Str("queue", q.channelPool.queue).
				Str("routingKey", q.channelPool.routingKey).
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
	poolSize := min(max(runtime.NumCPU()/4, 1), 5) // 连接池大小，32核CPU的机器最多创建5个连接
	if conn := newConnPool(urlStr, queueConfig.Address, poolSize); conn != nil {
		poolSize = poolSize * 10 // 通道池大小，32核CPU的机器最多创建50个通道
		pool := newChannelPool(conn, queueConfig, poolSize, dequeueSize)
		if pool == nil {
			return
		}
		connChannelPoolMap[connChannelPoolId] = pool
	}
}

// makeQueue 创建一个队列
func makeQueue(connPoolMap map[string]*channelPool, queueMap map[string]*Queue, queueConfig configs.AMQP091Queue, dequeueSize int) *Queue {
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

	// 使用 Address、Username  作为 key 查找 connChannelPool
	connChannelPoolId := fmt.Sprintf("Address%sUsername%s", queueConfig.Address, queueConfig.Username)
	conn := connPoolMap[connChannelPoolId]
	if conn == nil {
		return nil
	}
	q := newQueue(conn, *queueConfig.Durable, dequeueSize)
	queueMap[queueId] = q
	return q
}
