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
	internalMetrics "netsvr/internal/metrics"
	"netsvr/pkg/quit"
	"sync/atomic"
	"time"
)

type published struct {
	seqNo uint64 // 消息编号
	size  int    // 消息大小
}

type amqpChInfo struct {
	// AMQP Channel
	channel *amqp.Channel
	// 已发布的消息
	publishedCh chan published
	//引用计数
	refCount int32
}

// channelPool 管理 AMQP Channel
type channelPool struct {
	connPool    *connPool
	pool        chan *amqpChInfo
	size        chan struct{}
	waitTimeout time.Duration
	dequeueSize int // 批量发送消息的大小
}

// newChannelPool 创建 Channel 管理器
func newChannelPool(conn *connPool, poolSize int, dequeueSize int) *channelPool {
	c := &channelPool{
		connPool:    conn,
		size:        make(chan struct{}, poolSize),
		pool:        make(chan *amqpChInfo, poolSize),
		waitTimeout: time.Second * 3,
		dequeueSize: dequeueSize,
	}
	for i := 0; i < poolSize; i++ {
		c.size <- struct{}{}
	}
	socket := c.getAmqpChannel()
	if socket == nil {
		return nil
	}
	defer c.release(socket)
	go c.loopHeartbeat()
	return c
}

func (cm *channelPool) loopHeartbeat() {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().Stack().Err(nil).Any("panic", panicErr).
				Str("address", cm.connPool.address).
				Msg("AMQP091 channel heartbeat coroutine is closed")
		} else {
			log.Logger.Debug().
				Str("address", cm.connPool.address).
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
			Msg("AMQP091 create channel failed")
		return nil
	}
	// 启用 Publisher Confirm
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		log.Logger.Error().Err(err).
			Str("address", cm.connPool.address).
			Msg("AMQP091 enable channel confirm failed")
		return nil
	}

	ret := &amqpChInfo{
		channel:     ch,
		publishedCh: make(chan published, cm.dequeueSize*2),
	}
	go cm.monitor(ret)
	return ret
}

func (cm *channelPool) monitor(amqpChInfo *amqpChInfo) {
	// 记录已发布的消息大小
	publishedMp := make(map[uint64]int, cm.dequeueSize)
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().Stack().Err(nil).Any("panic", panicErr).
				Str("address", cm.connPool.address).
				Msg("AMQP091 channel monitor is closed")
		} else {
			log.Logger.Debug().
				Str("address", cm.connPool.address).
				Msg("AMQP091 channel monitor is closed")
		}
		failedCount := 0
		for _, size := range publishedMp {
			if size > 0 {
				//有消息大小的才是发送成功的消息
				failedCount++
			}
		}
		if failedCount > 0 {
			internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessFailedCount].Meter.Mark(int64(failedCount))
		}
	}()
	// 监听 channel是否关闭
	errCh := amqpChInfo.channel.NotifyClose(make(chan *amqp.Error, 1))
	// 监听 channel是否发布成功
	confirmCh := amqpChInfo.channel.NotifyPublish(make(chan amqp.Confirmation, cm.dequeueSize*2))
	for {
		select {
		case notify, ok := <-errCh:
			if ok {
				//mq服务器主动通知关闭
				log.Logger.Error().Err(notify).
					Str("address", cm.connPool.address).
					Msg("AMQP091 channel is closed")
			}
			goto delayEnd
		case pb := <-amqpChInfo.publishedCh:
			if pb.size > 0 {
				//记录已发布的消息大小
				publishedMp[pb.seqNo] = pb.size
			} else {
				//撤销已记录的消息大小
				delete(publishedMp, pb.seqNo)
			}
		case confirm, ok := <-confirmCh:
			if !ok {
				//confirmCh已经被close
				goto delayEnd
			}
			if confirm.Ack {
				size := publishedMp[confirm.DeliveryTag]
				internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessSucceedCount].Meter.Mark(1)
				internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessSucceedByte].Meter.Mark(int64(size))
			} else {
				internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessFailedCount].Meter.Mark(1)
			}
			delete(publishedMp, confirm.DeliveryTag)
		}
	}
delayEnd:
	//确保关闭channel
	amqpChInfo.channel.Close()
	//检测连接状态，将已经关闭的amqp channel 从连接池中移除
	cm.heartbeat()
	//延迟一段时间再结束协程，继续消费，确保publishedCh发送端不会因为没有消费者而死锁
	delay := time.NewTimer(time.Second * 5)
	defer func() {
		delay.Stop()
	}()
retry:
	for {
		select {
		case <-quit.Ctx.Done():
			return
		case pb := <-amqpChInfo.publishedCh:
			if pb.size > 0 {
				//记录已发布的消息大小
				publishedMp[pb.seqNo] = pb.size
			} else {
				//撤销已记录的消息大小
				delete(publishedMp, pb.seqNo)
			}
		case <-delay.C:
			if atomic.LoadInt32(&amqpChInfo.refCount) == 0 && len(amqpChInfo.publishedCh) == 0 {
				//没有活跃生产者，publishedCh中也没有残留，执行释放逻辑
				for {
					//confirmCh中可能还有数据，需要处理
					confirm, ok := <-confirmCh
					if !ok {
						break
					}
					if confirm.Ack {
						size := publishedMp[confirm.DeliveryTag]
						internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessSucceedCount].Meter.Mark(1)
						internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessSucceedByte].Meter.Mark(int64(size))
					} else {
						internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessFailedCount].Meter.Mark(1)
					}
					delete(publishedMp, confirm.DeliveryTag)
				}
				//释放逻辑结束
				return
			}
			//继续监听 publishedCh
			delay.Reset(time.Second * 5)
			goto retry
		}
	}
}

func (cm *channelPool) getAmqpChannel() *amqpChInfo {
	if len(cm.pool) == 0 {
		select {
		case <-cm.size:
			socket := cm.createChannel()
			if socket == nil || socket.channel.IsClosed() {
				// 等待3秒的时间再释放重连机会，否则会频繁创建连接
				time.Sleep(time.Second * 3)
				cm.size <- struct{}{}
				return nil
			} else {
				log.Logger.Info().
					Str("address", cm.connPool.address).
					Msg("AMQP091 establish new channel")
				//引用计数加1
				atomic.AddInt32(&socket.refCount, 1)
				return socket
			}
		default:
			goto wait
		}
	}
wait:
	if cm.waitTimeout == 0 {
		socket := <-cm.pool
		atomic.AddInt32(&socket.refCount, 1)
		return socket
	}
	timeout := time.NewTimer(cm.waitTimeout)
	defer timeout.Stop()
	select {
	case socket := <-cm.pool:
		atomic.AddInt32(&socket.refCount, 1)
		return socket
	case <-timeout.C:
		return nil
	}
}

func (cm *channelPool) release(socket *amqpChInfo) {
	if socket == nil {
		cm.size <- struct{}{}
		return
	}
	//引用计数减1
	atomic.AddInt32(&socket.refCount, -1)
	if socket.channel.IsClosed() {
		cm.size <- struct{}{}
		return
	}
	cm.pool <- socket
}
