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
	"netsvr/internal/log"
	"netsvr/pkg/quit"
	"time"
)

// connPool 管理 AMQP 连接和重连逻辑
type connPool struct {
	url         string // 完整 URL（含密码，仅用于连接）
	address     string // 服务器地址（用于日志，不含密码）
	pool        chan *amqp.Connection
	size        chan struct{}
	waitTimeout time.Duration
}

// newConnPool 创建连接管理器
func newConnPool(url string, address string, poolSize int) *connPool {
	c := &connPool{
		url:         url,
		address:     address,
		size:        make(chan struct{}, poolSize),
		pool:        make(chan *amqp.Connection, poolSize),
		waitTimeout: time.Second * 3,
	}
	for i := 0; i < poolSize; i++ {
		c.size <- struct{}{}
	}
	socket := c.getAmqpConnection()
	if socket == nil {
		return nil
	}
	defer c.release(socket)
	go c.loopHeartbeat()
	return c
}

func (cm *connPool) loopHeartbeat() {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().Stack().Err(nil).Any("panic", panicErr).
				Str("address", cm.address).
				Msg("AMQP091 connection heartbeat coroutine is closed")
		} else {
			log.Logger.Debug().Str("address", cm.address).Msg("AMQP091 connection heartbeat coroutine is closed")
		}
	}()
	ticker := time.NewTicker(time.Second * 30)
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

func (cm *connPool) heartbeat() {
	for i := len(cm.pool); i > 0; i-- {
		select {
		case <-quit.Ctx.Done():
			return
		case socket := <-cm.pool:
			if socket.IsClosed() == false {
				cm.pool <- socket
			} else {
				cm.release(nil)
				log.Logger.Error().Str("address", cm.address).Msg("AMQP091 connection is closed")
			}
		default:
			continue
		}
	}
}

func (cm *connPool) createAmpqConn() *amqp.Connection {
	socket, err := amqp.Dial(cm.url)
	if err != nil {
		log.Logger.Error().Err(err).Str("address", cm.address).Msg("AMQP091 dial failed")
		return nil
	}
	go func(socket *amqp.Connection) {
		for {
			notify, ok := <-socket.NotifyClose(make(chan *amqp.Error, 1))
			if ok {
				//mq服务器主动通知关闭
				log.Logger.Error().Err(notify).Str("address", cm.address).Msg("AMQP091 connection is closed")
			}
			//检测连接状态
			cm.heartbeat()
			return
		}
	}(socket)
	return socket
}

func (cm *connPool) getAmqpConnection() *amqp.Connection {
	if len(cm.pool) == 0 {
		select {
		case <-cm.size:
			socket := cm.createAmpqConn()
			if socket == nil || socket.IsClosed() {
				// 等待3秒的时间再释放重连机会，否则会频繁创建连接
				time.Sleep(time.Second * 3)
				cm.size <- struct{}{}
				return nil
			} else {
				log.Logger.Info().Str("address", cm.address).Msg("AMQP091 establish new connection")
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
		return nil
	}
}

func (cm *connPool) release(socket *amqp.Connection) {
	if socket == nil || socket.IsClosed() {
		cm.size <- struct{}{}
		return
	}
	cm.pool <- socket
}
