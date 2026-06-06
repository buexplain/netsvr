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
	"sync"
	"time"
)

// conn 管理 AMQP 连接和重连逻辑
type conn struct {
	url      string // 完整 URL（含密码，仅用于连接）
	address  string // 服务器地址（用于日志，不含密码）
	ampqConn *amqp.Connection
	mux      sync.RWMutex
	closes   []chan struct{}
}

// newConn 创建连接管理器
func newConn(url string, address string) *conn {
	c := &conn{
		url:     url,
		address: address,
		closes:  []chan struct{}{},
	}
	if c.connect() {
		// 启动监听协程
		go c.monitorConnection()
		return c
	}
	return nil
}

func (cm *conn) notifyClose(receiver chan struct{}) chan struct{} {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	cm.closes = append(cm.closes, receiver)
	return receiver
}

// connect 建立连接
func (cm *conn) connect() bool {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	//通知所有监听者
	for _, v := range cm.closes {
		select {
		case v <- struct{}{}:
			continue
		default:
			continue
		}
	}
	// 建立连接
	ampqConn, err := amqp.Dial(cm.url)
	if err != nil {
		log.Logger.Error().Err(err).Str("address", cm.address).Msg("AMQP091 dial failed")
		return false
	}
	cm.ampqConn = ampqConn
	log.Logger.Info().Str("address", cm.address).Msg("AMQP091 conn connected")
	return true
}

// monitorConnection 监听连接状态并处理重连
func (cm *conn) monitorConnection() {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().Stack().Err(nil).Any("panic", panicErr).
				Str("address", cm.address).Msg("AMQP091 monitor conn coroutine is closed")
		} else {
			log.Logger.Debug().Str("address", cm.address).Msg("AMQP091 monitor conn coroutine is closed")
		}
	}()
loop:
	for {
		notify, ok := <-cm.ampqConn.NotifyClose(make(chan *amqp.Error))
		if ok {
			log.Logger.Error().Err(notify).Str("address", cm.address).
				Msg("AMQP091 conn closed, will reconnect")
		} else {
			log.Logger.Error().Str("address", cm.address).
				Msg("AMQP091 conn closed, will reconnect")
		}
		//重连
		for {
			//检查是否退出
			select {
			case <-quit.Ctx.Done():
				return
			default:
				if cm.connect() {
					//重连成功，继续监听
					goto loop
				}
			}
			//休眠3秒
			time.Sleep(time.Second * 3)
		}
	}
}

func (cm *conn) getAmqpConnection() *amqp.Connection {
	cm.mux.RLock()
	defer cm.mux.RUnlock()
	if cm.ampqConn == nil || cm.ampqConn.IsClosed() {
		return nil
	}
	return cm.ampqConn
}
