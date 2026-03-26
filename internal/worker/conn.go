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
	"runtime"
	"sync/atomic"
	"time"
)

type Conn struct {
	conn      net.Conn
	sendCh    chan net.Buffers
	connId    string
	closeLock int32
	events    int32
}

func newConn(conn net.Conn) *Conn {
	tmp := &Conn{
		conn:   conn,
		connId: newConnId(conn.RemoteAddr().String()),
		sendCh: make(chan net.Buffers, configs.Config.Worker.SendChanCap),
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
		close(r.sendCh)
	})
	//丢弃所有数据，让所有的Send函数能正常写入数据，而不是报错：send on closed channel
	for range r.sendCh {
		continue
	}
}

func (r *Conn) loopSend() {
	defer func() {
		if err := recover(); err != nil {
			log.Logger.Error().
				Stack().Err(nil).
				Type("recoverType", err).
				Interface("recover", err).
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
	for data := range r.sendCh {
		r.send(data)
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
		r.formatSendToBusinessData(buffers[0], buffers[1], log.Logger.Error()).
			Err(err).
			Int32("events", r.GetEvents()).
			Str("connId", r.connId).
			Msg("Worker send failed and discard message")
		return
	}
	//其他错误或已写入部分数据：连接状态不可恢复，强制关闭
	r.Close()
	r.formatSendToBusinessData(buffers[0], buffers[1], log.Logger.Error()).
		Err(err).
		Int32("events", r.GetEvents()).
		Str("connId", r.connId).
		Msg("Worker send failed and force close conn")
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
				Stack().Err(nil).
				Type("recoverType", panicErr).
				Interface("recover", panicErr).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker send sendCh failed")
		}
	}()
	// 编码业务数据
	body, err := proto.Marshal(message)
	if err != nil {
		return 0
	}
	//填充 cmd 字段 (大端序)
	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header[4:8], uint32(cmd))
	//填充长度字段 (大端序)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(body)+4))
	nb := net.Buffers{
		header,
		body,
	}
	//发送出去
	if len(r.sendCh) < cap(r.sendCh) || configs.Config.Worker.SendChanDeadline == 0 {
		r.sendCh <- nb
		return 8 + len(body)
	}
	timeout := time.NewTimer(configs.Config.Worker.SendChanDeadline)
	select {
	case r.sendCh <- nb:
		timeout.Stop()
		return 8 + len(body)
	case <-timeout.C:
		timeout.Stop()
		//统计worker到business的失败次数
		internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessFailedCount].Meter.Mark(1)
		r.formatSendToBusinessData(header, body, log.Logger.Error()).Err(errors.New("send to blocking channel timeout")).
			Int32("events", r.GetEvents()).
			Str("connId", r.connId).
			Msg("Worker send failed and discard message")
		return 0
	}
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
