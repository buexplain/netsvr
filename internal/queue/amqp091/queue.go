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
	"sync/atomic"

	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/log"
	internalMetrics "netsvr/internal/metrics"
	"netsvr/internal/queue/internal"
	"netsvr/pkg/queue"
	"time"
)

// Queue AMQP队列
type Queue struct {
	closeLock    int32
	sendCh       *queue.Queue[*internal.Packet]
	channel      *channel // Channel 管理器
	dequeueSize  int
	deliveryMode uint8 // 1: 持久化，2: 临时
}

func newQueue(channel *channel, durable bool, dequeueSize int) *Queue {
	// 处理指针字段
	q := &Queue{
		sendCh:      queue.New[*internal.Packet](1024),
		channel:     channel,
		dequeueSize: dequeueSize,
	}
	if durable {
		q.deliveryMode = amqp.Persistent
	} else {
		q.deliveryMode = amqp.Transient
	}
	// 启动发送协程
	go q.loopSend()
	return q
}

func (q *Queue) loopSend() {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().Stack().Err(nil).Any("panic", panicErr).
				Str("address", q.channel.conn.address).
				Str("exchange", q.channel.exchange).
				Str("queue", q.channel.queue).
				Str("routingKey", q.channel.routingKey).
				Msg("AMQP091 send coroutine closed")
		} else {
			log.Logger.Debug().
				Str("address", q.channel.conn.address).
				Str("exchange", q.channel.exchange).
				Str("queue", q.channel.queue).
				Str("routingKey", q.channel.routingKey).
				Msg("AMQP091 send coroutine is closed")
		}
	}()
	packets := make([]*internal.Packet, q.dequeueSize)
	for {
		count := q.sendCh.Dequeue(packets)
		if count == 0 {
			return // 队列已关闭
		}
		packetsCopy := make([]*internal.Packet, count)
		for i := 0; i < count; i++ {
			packetsCopy[i] = packets[i]
			packets[i] = nil // 清空引用
		}
		if err := goroutine.DefaultWorkerPool.Submit(func() {
			defer func() {
				// 归还所有 Packet
				for _, pkg := range packetsCopy {
					internal.PacketObjPool.Put(pkg)
				}
			}()
			q.sendBatch(packetsCopy)
		}); err != nil {
			// 提交失败，立即归还
			for _, pkg := range packetsCopy {
				internal.PacketObjPool.Put(pkg)
			}
			internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessFailedCount].Meter.Mark(int64(count))
			log.Logger.Error().Err(err).
				Str("address", q.channel.conn.address).
				Str("exchange", q.channel.exchange).
				Str("queue", q.channel.queue).
				Str("routingKey", q.channel.routingKey).
				Msg("AMQP091 submit to worker pool failed")
		}
	}
}

// sendBatch 批量发送消息到 AMQP（在协程池中执行）
func (q *Queue) sendBatch(packets []*internal.Packet) {
	amqpChannel, confirmChan := q.channel.getAmqpChannel()
	if amqpChannel == nil {
		internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessSucceedCount].Meter.Mark(int64(len(packets)))
		log.Logger.Error().
			Str("address", q.channel.conn.address).
			Str("exchange", q.channel.exchange).
			Str("queue", q.channel.queue).
			Str("routingKey", q.channel.routingKey).
			Msg("AMQP091 channel not available")
		return
	}

	// 记录已发布的消息大小
	published := make(map[uint64]int, len(packets))

	// 批量发布消息
	for _, pkg := range packets {
		seqNo := amqpChannel.GetNextPublishSeqNo()
		// 发布消息
		err := amqpChannel.Publish(
			q.channel.exchange,
			q.channel.routingKey,
			false, // mandatory: 找不到队列时是否返回错误
			false, // immediate: 没有消费者时是否返回错误
			amqp.Publishing{
				ContentType:  "application/octet-stream",
				Body:         pkg.Message,
				DeliveryMode: q.deliveryMode,
			},
		)
		// 发布失败
		if err != nil {
			internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessFailedCount].Meter.Mark(1)
			log.Logger.Error().Err(err).
				Str("address", q.channel.conn.address).
				Str("exchange", q.channel.exchange).
				Str("queue", q.channel.queue).
				Str("routingKey", q.channel.routingKey).
				Msg("AMQP091 publish failed")
		} else {
			// 记录已发布的消息大小
			published[seqNo] = len(pkg.Message)
		}
	}

	// 等待所有消息的 ACK
	failedCount := 0  // 失败的消息数量
	succeedCount := 0 // 成功发布的消息数量
	succeedByte := 0  // 成功发布的消息大小
	for i := len(published); i > 0; i-- {
		confirm, ok := <-confirmChan
		if !ok {
			// Channel 已关闭
			log.Logger.Error().
				Str("address", q.channel.conn.address).
				Str("exchange", q.channel.exchange).
				Str("queue", q.channel.queue).
				Str("routingKey", q.channel.routingKey).
				Msg("AMQP091 confirm amqpChannel closed")
			failedCount += i
			break
		}
		if confirm.Ack {
			succeedByte += published[confirm.DeliveryTag]
			succeedCount++
		} else {
			failedCount++
		}
	}
	//统计指标
	if succeedCount > 0 {
		internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessSucceedCount].Meter.Mark(int64(succeedCount))
		internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessSucceedByte].Meter.Mark(int64(succeedByte))
	}
	if failedCount > 0 {
		internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessSucceedCount].Meter.Mark(int64(failedCount))
	}
}

// close 关闭队列
func (q *Queue) close() bool {
	defer func() {
		_ = recover()
	}()

	if !atomic.CompareAndSwapInt32(&q.closeLock, 0, 1) {
		return false
	}

	// 延迟关闭 sendCh，让 loopSend 处理完剩余消息
	time.AfterFunc(100*time.Millisecond, func() {
		q.sendCh.Close()
	})

	return true
}

// Send 发送消息到 AMQP 队列
func (q *Queue) Send(message proto.Message, cmd netsvrProtocol.Cmd) {
	var pkg *internal.Packet
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().Stack().Err(nil).Any("panic", panicErr).
				Str("address", q.channel.conn.address).
				Str("exchange", q.channel.exchange).
				Str("queue", q.channel.queue).
				Str("routingKey", q.channel.routingKey).
				Msg("AMQP091 send sendCh failed")
		}
		// 仍持有 pkg 时统一归还
		if pkg != nil {
			internal.PacketObjPool.Put(pkg)
		}
	}()
	pkg = internal.PacketObjPool.Get()
	err := pkg.Set(message, cmd)
	if err != nil {
		log.Logger.Error().Err(err).
			Str("address", q.channel.conn.address).
			Str("exchange", q.channel.exchange).
			Str("queue", q.channel.queue).
			Str("routingKey", q.channel.routingKey).
			Msg("AMQP091 proto.Marshal failed")
	}
	if q.sendCh.Enqueue(pkg) {
		pkg = nil // 所有权已转移
		return
	}
	internalMetrics.Registry[internalMetrics.ItemAMQP091ToBusinessFailedCount].Meter.Mark(1)
	log.Logger.Error().
		Str("address", q.channel.conn.address).
		Str("exchange", q.channel.exchange).
		Str("queue", q.channel.queue).
		Str("routingKey", q.channel.routingKey).
		Msg("AMQP091 send failed and discard message")
}
