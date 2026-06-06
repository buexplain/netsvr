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

// Package redisQueue Redis队列
package redisQueue

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/gobwas/ws"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/log"
	internalMetrics "netsvr/internal/metrics"
	"netsvr/internal/queue/internal"
	"netsvr/pkg/queue"
	"netsvr/pkg/quit"
	"sync/atomic"
	"time"
)

// Queue Redis队列
type Queue struct {
	closeLock   int32
	sendCh      *queue.Queue[*internal.Packet]
	redisClient *redis.Client
	redisKey    string
	redisDB     int
	keyType     string
	dequeueSize int // 一次从队列中取出的元素数量
}

func newQueue(redisClient *redis.Client, queueConfig configs.RedisQueue, dequeueSize int) *Queue {
	q := &Queue{
		redisClient: redisClient,
		redisKey:    queueConfig.Key,
		redisDB:     queueConfig.DB,
		keyType:     queueConfig.KeyType,
		sendCh:      queue.New[*internal.Packet](1024),
		dequeueSize: dequeueSize,
	}
	if queueConfig.KeyType == "list" {
		go q.loopSend(q.sendListBatchMode, q.sendListSingleMode)
	} else if queueConfig.KeyType == "stream" {
		go q.loopSend(q.sendStreamBatchMode, q.sendStreamSingleMode)
	} else {
		panic(fmt.Sprintf("redisQueue queue init failed %s", queueConfig.KeyType))
	}
	return q
}

func (r *Queue) Close() bool {
	defer func() {
		_ = recover()
	}()
	if !atomic.CompareAndSwapInt32(&r.closeLock, 0, 1) {
		return false
	}
	time.AfterFunc(time.Millisecond*100, func() {
		r.sendCh.Close()
	})
	return true
}

// 循环发送数据
func (r *Queue) loopSend(batchMode func(packets []*internal.Packet, size int), singleMode func(pkg *internal.Packet)) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().
				Stack().Err(nil).Any("panic", panicErr).
				Str("redisKey", r.redisKey).
				Str("keyType", r.keyType).
				Msg("RedisQueue send coroutine is closed")
		} else {
			log.Logger.Debug().
				Str("redisKey", r.redisKey).
				Str("keyType", r.keyType).
				Msg("RedisQueue send coroutine is closed")
		}
	}()
	packLimit := max(configs.Config.Customer.ReceivePackLimit, r.dequeueSize*2*1024)
	packets := make([]*internal.Packet, r.dequeueSize)
	var size int
	var i int
	var count int
	for {
		count = r.sendCh.Dequeue(packets)
		if count == 0 {
			return
		}
		size = 0
		for i = 0; i < count; i++ {
			//累计message大小
			size += len(packets[i].Message)
		}
		//整批数据小于单个数据包大小的限制，可以直接发送给redis
		if size < packLimit {
			packetsCopy := make([]*internal.Packet, count)
			for i = 0; i < count; i++ {
				packetsCopy[i] = packets[i]
				//清空
				packets[i] = nil
			}
			//发送
			sizeCopy := size //解决闭包引用导致的数据竞争
			if err := goroutine.DefaultWorkerPool.Submit(func() {
				defer func() {
					//归还
					for _, pkg := range packetsCopy {
						internal.PacketObjPool.Put(pkg)
					}
				}()
				batchMode(packetsCopy, sizeCopy)
			}); err != nil {
				//归还
				for _, pkg := range packetsCopy {
					internal.PacketObjPool.Put(pkg)
				}
				internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessFailedCount].Meter.Mark(1)
				log.Logger.Error().Err(err).
					Str("redisKey", r.redisKey).
					Str("keyType", r.keyType).
					Msg("RedisQueue submit to worker pool failed")
			}
		} else {
			//整批数据大于单个数据包大小的限制，改为循环单个发送，避免突破单个数据包限制的大小，给redis造成压力
			for i = 0; i < count; i++ {
				//发送
				pkg := packets[i]
				if err := goroutine.DefaultWorkerPool.Submit(func() {
					//归还
					defer internal.PacketObjPool.Put(pkg)
					singleMode(pkg)
				}); err != nil {
					//归还
					internal.PacketObjPool.Put(pkg)
					internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessFailedCount].Meter.Mark(1)
					log.Logger.Error().Err(err).
						Str("redisKey", r.redisKey).
						Str("keyType", r.keyType).
						Msg("RedisQueue submit to worker pool failed")
				}
				//清空
				packets[i] = nil
			}
		}
	}
}

func (r *Queue) Send(message proto.Message, cmd netsvrProtocol.Cmd) int {
	var pkg *internal.Packet
	defer func() {
		if panicErr := recover(); panicErr != nil {
			log.Logger.Error().
				Stack().Err(nil).Any("panic", panicErr).
				Str("redisKey", r.redisKey).
				Str("keyType", r.keyType).
				Msg("RedisQueue send sendCh failed")
		}
		// 仍持有 pkg 时统一归还：set 失败、入队失败、或 panic（入队成功须先把 pkg 置 nil，避免与 loopSendList 双重 Put）
		if pkg != nil {
			internal.PacketObjPool.Put(pkg)
		}
	}()
	// 编码业务数据
	pkg = internal.PacketObjPool.Get()
	err := pkg.Set(message, cmd)
	if err != nil {
		log.Logger.Error().Err(err).
			Str("redisKey", r.redisKey).
			Str("keyType", r.keyType).
			Msg("RedisQueue proto.Marshal failed")
		return 0
	}
	//发送出去
	n := len(pkg.Message) // 入队列前计算一下数据包大小，避免入队列后计算，产生数据竞争
	if r.sendCh.Enqueue(pkg) {
		pkg = nil // 所有权已转移到 loopSendList，避免 defer 重复归还
		return n
	}
	//写入失败：统计指标
	internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessFailedCount].Meter.Mark(1)
	r.formatSendToBusinessData(pkg.Message[0:4], pkg.Message[4:], log.Logger.Error()).Err(errors.New("send to blocking channel failed")).
		Str("redisKey", r.redisKey).
		Str("keyType", r.keyType).
		Msg("RedisQueue send failed and discard message")
	return 0
}

// sendListBatchMode 批量数据发送
func (r *Queue) sendListBatchMode(packets []*internal.Packet, size int) {
	//发送
	pipe := r.redisClient.Pipeline()
	for _, pkg := range packets {
		pipe.LPush(quit.Ctx, r.redisKey, pkg.Message)
	}
	cmderList, err := pipe.Exec(quit.Ctx)
	if err == nil {
		//写入成功：统计指标
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Mark(int64(len(packets)))
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Mark(int64(size))
		return
	}
	//写入失败：统计指标
	log.Logger.Error().Err(err).
		Str("redisKey", r.redisKey).
		Str("keyType", r.keyType).
		Msg("RedisQueue send redisQueue failed")
	var failedCount int64
	var succeedSize int
	for j, cmder := range cmderList {
		if cmder.Err() == nil {
			succeedSize += len(packets[j].Message)
		} else {
			failedCount++
		}
	}
	internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessFailedCount].Meter.Mark(failedCount)
	if succeedSize > 0 {
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Mark(int64(len(packets)) - failedCount)
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Mark(int64(succeedSize))
	}
}

// sendListSingleMode 单个数据发送
func (r *Queue) sendListSingleMode(pkg *internal.Packet) {
	//发送到redis
	if err := r.redisClient.LPush(quit.Ctx, r.redisKey, pkg.Message).Err(); err != nil {
		//写入失败：统计指标
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessFailedCount].Meter.Mark(1)
		log.Logger.Error().Err(err).
			Str("redisKey", r.redisKey).
			Str("keyType", r.keyType).
			Msg("RedisQueue send redisQueue failed")
	} else {
		//写入成功：统计指标
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Mark(1)
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Mark(int64(len(pkg.Message)))
	}
}

// sendStreamBatchMode 发送批量数据
func (r *Queue) sendStreamBatchMode(packets []*internal.Packet, size int) {
	//发送
	pipe := r.redisClient.Pipeline()
	for _, pkg := range packets {
		args := &redis.XAddArgs{
			Stream: r.redisKey,
			Values: map[string]interface{}{
				"data": pkg.Message,
			},
		}
		pipe.XAdd(quit.Ctx, args)
	}
	cmderList, err := pipe.Exec(quit.Ctx)
	if err == nil {
		//写入成功：统计指标
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Mark(int64(len(packets)))
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Mark(int64(size))
		return
	}
	//写入失败：统计指标
	log.Logger.Error().Err(err).
		Str("redisKey", r.redisKey).
		Str("keyType", r.keyType).
		Msg("RedisQueue send redisQueue failed")
	var failedCount int64
	var succeedSize int
	for j, cmder := range cmderList {
		if cmder.Err() == nil {
			succeedSize += len(packets[j].Message)
		} else {
			failedCount++
		}
	}
	internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessFailedCount].Meter.Mark(failedCount)
	if succeedSize > 0 {
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Mark(int64(len(packets)) - failedCount)
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Mark(int64(succeedSize))
	}
}

// sendStreamSingleMode 单个数据发送
func (r *Queue) sendStreamSingleMode(pkg *internal.Packet) {
	//发送到redis
	args := &redis.XAddArgs{
		Stream: r.redisKey,
		Values: map[string]interface{}{
			"data": pkg.Message,
		},
	}
	if err := r.redisClient.XAdd(quit.Ctx, args).Err(); err != nil {
		//写入失败：统计指标
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessFailedCount].Meter.Mark(1)
		log.Logger.Error().Err(err).
			Str("redisKey", r.redisKey).
			Str("keyType", r.keyType).
			Msg("RedisQueue send redisQueue failed")
	} else {
		//写入成功：统计指标
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Mark(1)
		internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Mark(int64(len(pkg.Message)))
	}
}

func (r *Queue) formatSendToBusinessData(cmdBytes []byte, body []byte, event *zerolog.Event) *zerolog.Event {
	cmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(cmdBytes))
	if cmd == netsvrProtocol.Cmd_Transfer {
		tf := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(body, tf); err != nil {
			return event
		}
		event = event.Str("cmd", cmd.String()).Str("uniqId", tf.UniqId).
			Str("session", tf.Session).
			Str("customerId", tf.CustomerId).
			Strs("topics", tf.Topics)
		if configs.Config.Customer.SendMessageType == ws.OpText {
			return event.Str("data", string(tf.Data))
		}
		return event.Hex("dataHex", tf.Data)
	}
	if cmd == netsvrProtocol.Cmd_ConnOpen {
		co := &netsvrProtocol.ConnOpen{}
		if err := proto.Unmarshal(body, co); err != nil {
			return event
		}
		return event.Str("cmd", cmd.String()).Str("uniqId", co.UniqId).
			Str("rawQuery", co.RawQuery).
			Str("xForwardedFor", co.XForwardedFor).
			Str("xRealIp", co.XRealIp).
			Str("remoteAddr", co.RemoteAddr)
	}
	if cmd == netsvrProtocol.Cmd_ConnClose {
		cc := &netsvrProtocol.ConnClose{}
		if err := proto.Unmarshal(body, cc); err != nil {
			return event
		}
		return event.Str("cmd", cmd.String()).Str("uniqId", cc.UniqId).
			Str("customerId", cc.CustomerId).
			Str("session", cc.Session).
			Strs("topics", cc.Topics)
	}
	//非客户端的命令，只打印cmd
	return event.Str("cmd", cmd.String())
}
