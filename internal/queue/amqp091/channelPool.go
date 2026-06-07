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

package amqp091

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"netsvr/configs"
	"netsvr/internal/log"
	"netsvr/pkg/quit"
	"time"
)

type amqpChInfo struct {
	channel *amqp.Channel
	confirm chan amqp.Confirmation
}

// channelPool 管理 AMQP Channel
type channelPool struct {
	connPool     *connPool
	exchange     string
	exchangeType string
	queue        string
	routingKey   string
	autoDelete   bool // 是否自动删除
	dequeueSize  int
	pool         chan *amqpChInfo
	size         chan struct{}
	waitTimeout  time.Duration
}

// newChannelPool 创建 Channel 管理器
func newChannelPool(conn *connPool, config configs.AMQP091Queue, poolSize int, dequeueSize int) *channelPool {
	c := &channelPool{
		connPool:     conn,
		exchange:     config.Exchange,
		exchangeType: config.ExchangeType,
		queue:        *config.Queue,
		routingKey:   *config.RoutingKey,
		dequeueSize:  dequeueSize,
		size:         make(chan struct{}, poolSize),
		pool:         make(chan *amqpChInfo, poolSize),
		waitTimeout:  time.Second * 3,
	}
	for i := 0; i < poolSize; i++ {
		c.size <- struct{}{}
	}
	socket := c.getAmqpChannel()
	if socket == nil {
		return nil
	}
	defer c.release(socket)
	if c.topologyDeclare(socket.channel) {
		go c.loopHeartbeat()
		return c
	}
	return nil
}

func (cm *channelPool) loopHeartbeat() {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().Stack().Err(nil).Any("panic", panicErr).
				Str("address", cm.connPool.address).
				Str("exchange", cm.exchange).
				Str("queue", cm.queue).
				Str("routingKey", cm.routingKey).
				Msg("AMQP091 channel heartbeat coroutine is closed")
		} else {
			log.Logger.Debug().
				Str("address", cm.connPool.address).
				Str("exchange", cm.exchange).
				Str("queue", cm.queue).
				Str("routingKey", cm.routingKey).
				Msg("AMQP091 channel heartbeat coroutine is closed")
		}
	}()
	ticker := time.NewTicker(time.Second * 25)
	defer ticker.Stop()
	for {
		select {
		case <-quit.Ctx.Done():
			return
		case <-ticker.C:
			cm.heartbeat()
		}
	}
}

func (cm *channelPool) heartbeat() {
	for i := len(cm.pool); i > 0; i-- {
		select {
		case <-quit.Ctx.Done():
			return
		case socket := <-cm.pool:
			if socket.channel.IsClosed() == false {
				cm.pool <- socket
			} else {
				cm.release(nil)
				log.Logger.Error().
					Str("address", cm.connPool.address).
					Str("exchange", cm.exchange).
					Str("queue", cm.queue).
					Str("routingKey", cm.routingKey).
					Msg("AMQP091 channel is closed.")
			}
		default:
			continue
		}
	}
}

func (cm *channelPool) createChannel() *amqpChInfo {
	amqpConn := cm.connPool.getAmqpConnection()
	if amqpConn == nil {
		return nil
	}
	defer cm.connPool.release(amqpConn)
	ch, err := amqpConn.Channel()
	if err != nil {
		log.Logger.Error().Err(err).
			Str("address", cm.connPool.address).
			Str("exchange", cm.exchange).
			Str("queue", cm.queue).
			Str("routingKey", cm.routingKey).
			Msg("AMQP091 create channelPool failed")
		return nil
	}
	// 启用 Publisher Confirm
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		log.Logger.Error().Err(err).
			Str("address", cm.connPool.address).
			Str("exchange", cm.exchange).
			Str("queue", cm.queue).
			Str("routingKey", cm.routingKey).
			Msg("AMQP091 enable channelPool confirm failed")
		return nil
	}

	ret := &amqpChInfo{
		channel: ch,
		confirm: ch.NotifyPublish(make(chan amqp.Confirmation, cm.dequeueSize)),
	}

	go func(ch *amqp.Channel) {
		for {
			notify, ok := <-ch.NotifyClose(make(chan *amqp.Error, 1))
			if ok {
				//mq服务器主动通知关闭
				log.Logger.Error().Err(notify).
					Str("address", cm.connPool.address).
					Str("exchange", cm.exchange).
					Str("queue", cm.queue).
					Str("routingKey", cm.routingKey).
					Msg("AMQP091 channel is closed")
			}
			//检测连接状态
			cm.heartbeat()
			return
		}
	}(ret.channel)
	return ret
}

func (cm *channelPool) getAmqpChannel() *amqpChInfo {
	if len(cm.pool) == 0 {
		select {
		case <-cm.size:
			socket := cm.createChannel()
			if socket == nil || socket.channel.IsClosed() {
				cm.size <- struct{}{}
				log.Logger.Error().
					Str("address", cm.connPool.address).
					Str("exchange", cm.exchange).
					Str("queue", cm.queue).
					Str("routingKey", cm.routingKey).
					Msg("AMQP091 cannot establish new channel")
				return nil
			} else {
				log.Logger.Info().
					Str("address", cm.connPool.address).
					Str("exchange", cm.exchange).
					Str("queue", cm.queue).
					Str("routingKey", cm.routingKey).
					Msg("AMQP091 establish new channel")
				return socket
			}
		default:
			goto wait
		}
	}
wait:
	if cm.waitTimeout == 0 {
		return <-cm.pool
	}
	timeout := time.NewTimer(cm.waitTimeout)
	defer timeout.Stop()
	select {
	case socket := <-cm.pool:
		return socket
	case <-timeout.C:
		log.Logger.Error().
			Str("address", cm.connPool.address).
			Str("exchange", cm.exchange).
			Str("queue", cm.queue).
			Str("routingKey", cm.routingKey).
			Msg("AMQP091 cannot establish new channel before wait_timeout")
		return nil
	}
}

func (cm *channelPool) release(socket *amqpChInfo) {
	if socket == nil || socket.channel.IsClosed() {
		cm.size <- struct{}{}
		return
	}
	cm.pool <- socket
}

// topologyDeclare 声明 AMQP 拓扑结构（Exchange、Queue、Binding）
func (cm *channelPool) topologyDeclare(amqpChannel *amqp.Channel) bool {
	// 声明 Exchange
	if err := amqpChannel.ExchangeDeclare(
		cm.exchange,
		cm.exchangeType,
		true,          // durable: 持久化交换机
		cm.autoDelete, // autoDelete: 自动删除
		false,         // internal: 内部交换机
		false,         // noWait: 不等待响应
		nil,           // arguments
	); err != nil {
		log.Logger.Error().Err(err).
			Str("address", cm.connPool.address).
			Str("exchange", cm.exchange).
			Str("queue", cm.queue).
			Str("routingKey", cm.routingKey).
			Msg("AMQP091 declare exchange failed")
		return false
	}

	// 如果配置了 Queue 名称，则声明队列并绑定
	if cm.queue != "" {
		_, err := amqpChannel.QueueDeclare(
			cm.queue,
			true,          // durable: 持久化队列
			cm.autoDelete, // autoDelete: 自动删除
			false,         // exclusive: 非独占
			false,         // noWait: 不等待响应
			nil,           // arguments
		)
		if err != nil {
			log.Logger.Error().Err(err).
				Str("address", cm.connPool.address).
				Str("exchange", cm.exchange).
				Str("queue", cm.queue).
				Str("routingKey", cm.routingKey).
				Msg("AMQP091 declare queue failed")
			return false
		}

		if err := amqpChannel.QueueBind(
			cm.queue,
			cm.routingKey,
			cm.exchange,
			false, // noWait: 不等待响应
			nil,   // arguments
		); err != nil {
			log.Logger.Error().Err(err).
				Str("address", cm.connPool.address).
				Str("exchange", cm.exchange).
				Str("queue", cm.queue).
				Str("routingKey", cm.routingKey).
				Msg("AMQP091 bind queue failed")
			return false
		}
	}

	log.Logger.Info().
		Str("address", cm.connPool.address).
		Str("exchange", cm.exchange).
		Str("queue", cm.queue).
		Str("routingKey", cm.routingKey).
		Msg("AMQP091 topology declared")

	return true
}
