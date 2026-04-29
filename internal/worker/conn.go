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

package worker

import (
	"encoding/binary"
	"errors"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/gobwas/ws"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"net"
	"netsvr/configs"
	"netsvr/internal/log"
	internalMetrics "netsvr/internal/metrics"
	"netsvr/pkg/queue"
	"runtime"
	"sync/atomic"
	"time"
)

type Conn struct {
	conn      net.Conn
	sendCh    *queue.Queue[*packet]
	connId    string
	closeLock int32
	events    int32
}

func newConn(conn net.Conn) *Conn {
	tmp := &Conn{
		conn:   conn,
		connId: newConnId(conn.RemoteAddr().String()),
		sendCh: queue.New[*packet](configs.Config.Worker.SendChanCap),
	}
	go tmp.loopSend()
	return tmp
}

func (r *Conn) Close() {
	defer func() {
		_ = recover()
	}()
	if !atomic.CompareAndSwapInt32(&r.closeLock, 0, 1) {
		return
	}
	Manager.Del(r.connId)
	_ = r.conn.Close()
	time.AfterFunc(time.Millisecond*100, func() {
		r.sendCh.Close()
	})
}

func (r *Conn) loopSend() {
	defer func() {
		if err := recover(); err != nil {
			log.Logger.Error().
				Stack().
				Any("panic", err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker send coroutine is closed")
		} else {
			log.Logger.Debug().
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker send coroutine is closed")
		}
	}()
	packLimit := max(configs.Config.Customer.ReceivePackLimit, 65536)
	var size int
	var i int
	var count int
	var pkg *packet
	packets := make([]*packet, 20) //假设一个用户消息是2kb，20个消息则会一次性写入40kb数据
	bulkBuffer := make(net.Buffers, 40)
	var length int // bulkBuffer的实际长度
	for {
		count = r.sendCh.Dequeue(packets)
		if count == 0 {
			return
		}
		size = count * 8
		length = 0
		for i = 0; i < count; i++ {
			//单个消息包
			pkg = packets[i]
			//累计body大小
			size += len(pkg.body)
			//构造成net.Buffers
			bulkBuffer[length] = pkg.header
			length++
			bulkBuffer[length] = pkg.body
			length++
			packetObjPool.Put(pkg) //回收
			packets[i] = nil       // 回收
		}
		//整批数据小于单个数据包大小的限制，可以直接发送给business
		if size < packLimit {
			r.send(bulkBuffer)
		} else {
			//整批数据大于单个数据包大小的限制，改为循环单个发送，避免突破单个数据包限制的大小，给business侧造成压力
			for i = 0; i < length; i += 2 {
				r.send(bulkBuffer[i : i+2])
			}
		}
		//回收bulkBuffer
		for i = 0; i < length; i++ {
			bulkBuffer[i] = nil
		}
	}
}

func (r *Conn) send(buffers net.Buffers) {
	if err := r.conn.SetWriteDeadline(time.Now().Add(configs.Config.Worker.SendDeadline)); err != nil {
		r.Close()
		log.Logger.Error().Err(err).
			Int32("events", r.GetEvents()).
			Str("connId", r.connId).
			Msg("Worker SetWriteDeadline failed")
		return
	}
	writeLen, err := buffers.WriteTo(r.conn)
	if err == nil {
		return
	}
	//写入失败：统计指标
	internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessFailedCount].Meter.Mark(1)
	if writeLen == 0 {
		for i := 0; i < len(buffers); {
			r.formatSendToBusinessData(buffers[i], buffers[i+1], log.Logger.Error()).
				Err(err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker send failed and discard message")
			i += 2
		}
		return
	}
	//其他错误或已写入部分数据：连接状态不可恢复，强制关闭
	r.Close()
	for i := 0; i < len(buffers); {
		r.formatSendToBusinessData(buffers[i], buffers[i+1], log.Logger.Error()).
			Err(err).
			Int32("events", r.GetEvents()).
			Str("connId", r.connId).
			Msg("Worker send failed and force close conn")
		i += 2
	}
}

func (r *Conn) GetConnRemoteAddr() string {
	return r.conn.RemoteAddr().String()
}

func (r *Conn) GetConnId() string {
	return r.connId
}

// GetEvents 返回business进程可以接收的events
func (r *Conn) GetEvents() int32 {
	return atomic.LoadInt32(&r.events)
}

// SetEvents 设置business进程可以接收的events
func (r *Conn) SetEvents(id int32) {
	atomic.StoreInt32(&r.events, id)
}

func (r *Conn) Send(message proto.Message, cmd netsvrProtocol.Cmd) int {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			var logEvent *zerolog.Event
			if runtimeError, ok := panicErr.(runtime.Error); ok && runtimeError.Error() == "send on closed channel" {
				//这个错误是无解的，因为正常情况下，channel的关闭是在生产者协程进行的
				//但是现在这里的生产者是多个，并且现在是读取或者是写失败产生的关闭，这里没有关闭的理由
				//所以只能降低日志级别，生产环境无需在意这个日志
				logEvent = log.Logger.Debug()
			} else {
				logEvent = log.Logger.Error()
			}
			logEvent.
				Stack().
				Any("panic", panicErr).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker send sendCh failed")
		}
	}()
	// 编码业务数据
	body, err := proto.Marshal(message)
	if err != nil {
		log.Logger.Error().Err(err).
			Int32("events", r.GetEvents()).
			Str("connId", r.connId).
			Msg("Worker proto.Marshal failed")
		return 0
	}
	//填充 cmd 字段 (大端序)
	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header[4:8], uint32(cmd))
	//填充长度字段 (大端序)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(body)+4))
	//发送出去
	pkg := packetObjPool.Get()
	pkg.header = header
	pkg.body = body
	if r.sendCh.Enqueue(pkg) {
		return 8 + len(body)
	}
	packetObjPool.Put(pkg)
	//统计worker到business的失败次数
	internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessFailedCount].Meter.Mark(1)
	r.formatSendToBusinessData(header, body, log.Logger.Error()).Err(errors.New("send to blocking channel failed")).
		Int32("events", r.GetEvents()).
		Str("connId", r.connId).
		Msg("Worker send failed and discard message")
	return 0
}

func (r *Conn) formatSendToBusinessData(header []byte, body []byte, event *zerolog.Event) *zerolog.Event {
	cmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(header[4:8]))
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
		co.GetUniqId()
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
