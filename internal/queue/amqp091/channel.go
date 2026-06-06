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
	"sync"
	"time"
)

// channel 管理 AMQP Channel
type channel struct {
	conn         *conn
	exchange     string
	exchangeType string
	queue        string
	routingKey   string
	autoDelete   bool // 是否自动删除
	amqpChannel  *amqp.Channel
	confirmCh    chan amqp.Confirmation
	mux          sync.RWMutex // Channel 的读写锁
}

// newChannel 创建 Channel 管理器
func newChannel(conn *conn, config configs.AMQP091Queue) *channel {
	ch := &channel{
		conn:         conn,
		mux:          sync.RWMutex{},
		exchange:     config.Exchange,
		exchangeType: config.ExchangeType,
		queue:        *config.Queue,
		routingKey:   *config.RoutingKey,
	}
	if ch.createChannel() {
		go ch.monitorConnection()
		return ch
	}
	return nil
}

// monitorConnection 监听连接状态并处理重连
func (cm *channel) monitorConnection() {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().Stack().Err(nil).Any("panic", panicErr).
				Str("address", cm.conn.address).
				Str("exchange", cm.exchange).
				Str("queue", cm.queue).
				Str("routingKey", cm.routingKey).
				Msg("AMQP091 monitor channel coroutine is closed")
		} else {
			log.Logger.Debug().
				Str("address", cm.conn.address).
				Str("exchange", cm.exchange).
				Str("queue", cm.queue).
				Str("routingKey", cm.routingKey).
				Msg("AMQP091 monitor channel coroutine is closed")
		}
	}()
	notifyClose := cm.conn.notifyClose(make(chan struct{}))
loop:
	for {
		<-notifyClose
		log.Logger.Error().
			Str("address", cm.conn.address).
			Str("exchange", cm.exchange).
			Str("queue", cm.queue).
			Str("routingKey", cm.routingKey).
			Msg("AMQP091 channel closed, will reconnect")
		//重新创建channel
		for {
			//检查是否退出
			select {
			case <-quit.Ctx.Done():
				return
			default:
				if cm.createChannel() {
					goto loop
				}
			}
			//休眠3秒
			time.Sleep(time.Second * 3)
		}
	}
}

// createChannel 创建 Channel、声明 AMQP 拓扑结构（Exchange、Queue、Binding）
func (cm *channel) createChannel() bool {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	amqpConn := cm.conn.getAmqpConnection()
	if amqpConn == nil {
		log.Logger.Error().
			Str("address", cm.conn.address).
			Str("exchange", cm.exchange).
			Str("queue", cm.queue).
			Str("routingKey", cm.routingKey).
			Msg("AMQP091 conn not available")
		return false
	}

	amqpChannel, err := amqpConn.Channel()
	if err != nil {
		log.Logger.Error().Err(err).
			Str("address", cm.conn.address).
			Str("exchange", cm.exchange).
			Str("queue", cm.queue).
			Str("routingKey", cm.routingKey).
			Msg("AMQP091 create channel failed")
		return false
	}

	// 启用 Publisher Confirm
	if err := amqpChannel.Confirm(false); err != nil {
		_ = amqpChannel.Close()
		log.Logger.Error().Err(err).
			Str("address", cm.conn.address).
			Str("exchange", cm.exchange).
			Str("queue", cm.queue).
			Str("routingKey", cm.routingKey).
			Msg("AMQP091 enable channel confirm failed")
		return false
	}

	log.Logger.Info().
		Str("address", cm.conn.address).
		Str("exchange", cm.exchange).
		Str("queue", cm.queue).
		Str("routingKey", cm.routingKey).
		Msg("AMQP091 channel created")

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
			Str("address", cm.conn.address).
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
				Str("address", cm.conn.address).
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
				Str("address", cm.conn.address).
				Str("exchange", cm.exchange).
				Str("queue", cm.queue).
				Str("routingKey", cm.routingKey).
				Msg("AMQP091 bind queue failed")
			return false
		}
	}

	cm.amqpChannel = amqpChannel
	cm.confirmCh = amqpChannel.NotifyPublish(make(chan amqp.Confirmation))

	log.Logger.Info().
		Str("address", cm.conn.address).
		Str("exchange", cm.exchange).
		Str("queue", cm.queue).
		Str("routingKey", cm.routingKey).
		Msg("AMQP091 topology declared")

	return true
}

func (cm *channel) getAmqpChannel() (*amqp.Channel, chan amqp.Confirmation) {
	cm.mux.RLock()
	defer cm.mux.RUnlock()
	if cm.amqpChannel == nil || cm.amqpChannel.IsClosed() {
		return nil, nil
	}
	return cm.amqpChannel, cm.confirmCh
}
