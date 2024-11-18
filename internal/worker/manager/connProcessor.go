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

package manager

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/configs"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/worker/connIdGen"
	"netsvr/pkg/quit"
	"runtime"
	"sync/atomic"
	"time"
)

type CmdCallback func(data []byte, processor *ConnProcessor)

type ConnProcessor struct {
	//business与worker的连接
	conn      net.Conn
	connId    string
	closeLock *int32
	//要发送给连接的数据
	sendCh chan []byte
	//从连接中读取的数据
	receiveCh chan []byte
	//当前连接可接收处理的事件
	events int32
	//各种命令的回调函数
	cmdCallback map[netsvrProtocol.Cmd]CmdCallback
}

func NewConnProcessor(conn net.Conn) *ConnProcessor {
	return &ConnProcessor{
		conn:        conn,
		connId:      connIdGen.New(conn.RemoteAddr().String()),
		closeLock:   new(int32),
		sendCh:      make(chan []byte, configs.Config.Worker.SendChanCap),
		receiveCh:   make(chan []byte, 64),
		events:      0,
		cmdCallback: map[netsvrProtocol.Cmd]CmdCallback{},
	}
}

func (r *ConnProcessor) GetConnRemoteAddr() string {
	return r.conn.RemoteAddr().String()
}

func (r *ConnProcessor) GetConnId() string {
	return r.connId
}

// ForceClose 强制关闭
func (r *ConnProcessor) ForceClose() {
	defer func() {
		if err := recover(); err != nil {
			log.Logger.Error().
				Stack().Err(nil).
				Type("recoverType", err).
				Interface("recover", err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker conn force close failed")
		}
	}()
	if !atomic.CompareAndSwapInt32(r.closeLock, 0, 1) {
		return
	}
	Manager.Del(r.connId)
	select {
	case <-quit.Ctx.Done():
	default:
		//因为连接本身的问题，强制关闭
		Shutter.Del(r)
	}
	time.AfterFunc(time.Millisecond*100, func() {
		close(r.sendCh)
	})
	//丢弃所有数据，让所有的Send函数能正常写入数据，而不是报错：send on closed channel
	for range r.sendCh {
		continue
	}
	_ = r.conn.Close()
}

func (r *ConnProcessor) LoopSend() {
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

func (r *ConnProcessor) send(data []byte) {
	var err error
	var writeLen int
	dataRef := data
	//写入到连接中
	for {
		//设置写超时
		if err = r.conn.SetWriteDeadline(time.Now().Add(configs.Config.Worker.SendDeadline)); err != nil {
			r.ForceClose()
			r.formatSendToBusinessDataData(dataRef, log.Logger.Error()).Err(err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker SetWriteDeadline to business conn failed")
			return
		}
		//写入数据
		writeLen, err = r.conn.Write(data)
		//写入成功
		if err == nil {
			//写入成功
			if writeLen == len(data) {
				return
			}
			//短写，继续写入
			data = data[writeLen:]
			continue
		}
		//写入错误
		//统计worker到business的失败次数
		metrics.Registry[metrics.ItemWorkerToBusinessFailedCount].Meter.Mark(1)
		//没有写入任何数据，tcp管道未被污染，丢弃本次数据，并打印日志
		var opErr *net.OpError
		if errors.As(err, &opErr) && opErr.Timeout() && len(dataRef) == len(data[writeLen:]) {
			r.formatSendToBusinessDataData(dataRef, log.Logger.Error()).Err(err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker send to business failed and discard message")
			return
		}
		//写入过部分数据，tcp管道已污染，对端已经无法拆包，必须关闭连接
		r.ForceClose()
		r.formatSendToBusinessDataData(dataRef, log.Logger.Error()).Err(err).
			Int32("events", r.GetEvents()).
			Str("connId", r.connId).
			Msg("Worker send to business failed and force close conn")
		return
	}
}

func (r *ConnProcessor) formatSendToBusinessDataData(data []byte, event *zerolog.Event) *zerolog.Event {
	cmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(data[4:8]))
	if cmd == netsvrProtocol.Cmd_Transfer {
		tf := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(data[8:], tf); err != nil {
			return event
		}
		event = event.Str("cmd", cmd.String()).Str("uniqId", tf.UniqId).
			Str("session", tf.Session).
			Str("customerId", tf.CustomerId).
			Strs("topics", tf.Topics)
		if configs.Config.Customer.SendMessageType == websocket.TextMessage {
			return event.Str("data", string(tf.Data))
		}
		return event.Hex("dataHex", tf.Data)
	}
	if cmd == netsvrProtocol.Cmd_ConnOpen {
		co := &netsvrProtocol.ConnOpen{}
		if err := proto.Unmarshal(data[8:], co); err != nil {
			return event
		}
		co.GetUniqId()
		return event.Str("cmd", cmd.String()).Str("uniqId", co.UniqId).
			Str("rawQuery", co.RawQuery).
			Strs("subProtocol", co.SubProtocol).
			Str("xForwardedFor", co.XForwardedFor).
			Str("xRealIp", co.XRealIp).
			Str("remoteAddr", co.RemoteAddr)
	}
	if cmd == netsvrProtocol.Cmd_ConnClose {
		cc := &netsvrProtocol.ConnClose{}
		if err := proto.Unmarshal(data[8:], cc); err != nil {
			return event
		}
		return event.Str("cmd", cmd.String()).Str("uniqId", cc.UniqId).
			Str("customerId", cc.CustomerId).
			Str("session", cc.Session).
			Strs("topics", cc.Topics)
	}
	return event
}

func (r *ConnProcessor) Send(message proto.Message, cmd netsvrProtocol.Cmd) int {
	defer func() {
		if err := recover(); err != nil {
			var logEvent *zerolog.Event
			if runtimeError, ok := err.(runtime.Error); ok && runtimeError.Error() == "send on closed channel" {
				//这个错误是无解的，因为正常情况下，channel的关闭是在生产者协程进行的
				//但是现在这里的生产者是多个，并且现在是读取或者是写失败产生的关闭，这里没有关闭的理由
				//所以只能降低日志级别，生产环境无需在意这个日志
				logEvent = log.Logger.Debug()
			} else {
				logEvent = log.Logger.Error()
			}
			logEvent.
				Stack().Err(nil).
				Type("recoverType", err).
				Interface("recover", err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker send sendCh failed")
		}
	}()
	data := make([]byte, 8)
	//先写业务层的cmd
	binary.BigEndian.PutUint32(data[4:8], uint32(cmd))
	if message == nil {
		//再写包头
		binary.BigEndian.PutUint32(data[0:4], 4)
		//发送出去
		r.sendCh <- data
		return 8
	}
	//再编码数据
	var err error
	pm := proto.MarshalOptions{}
	data, err = pm.MarshalAppend(data, message)
	if err == nil {
		//再写包头
		binary.BigEndian.PutUint32(data[0:4], uint32(len(data)-4))
		//发送出去
		r.sendCh <- data
		return len(data)
	}
	return 0
}

func (r *ConnProcessor) LoopReceive() {
	defer func() {
		if err := recover(); err != nil {
			log.Logger.Error().
				Stack().Err(nil).
				Type("recoverType", err).
				Interface("recover", err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker receive coroutine is closed")
		} else {
			log.Logger.Debug().
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker receive coroutine is closed")
		}
		close(r.receiveCh)
		r.ForceClose()
	}()
	//包头专用
	dataLenBuf := make([]byte, 4)
	var err error
	var dataLen uint32
	connReader := bufio.NewReaderSize(r.conn, configs.Config.Worker.ReadBufferSize)
	for {
		//设置读超时时间，再这个时间之内，business没有发数据过来，则会发生超时错误，导致连接被关闭
		if err = r.conn.SetReadDeadline(time.Now().Add(configs.Config.Worker.ReadDeadline)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker SetReadDeadline to business conn failed")
			break
		}
		//获取前4个字节，确定数据包长度
		dataLenBuf = dataLenBuf[:4]
		if _, err = io.ReadAtLeast(connReader, dataLenBuf, 4); err != nil {
			//读失败了，直接干掉这个连接，让business端重新连接进来，因为缓冲区的tcp流已经脏了，程序无法拆包
			//关掉重来，是最好的办法
			r.ForceClose()
			break
		}
		//这里采用大端序
		dataLen = binary.BigEndian.Uint32(dataLenBuf)
		//发送是数据包太大，直接关闭business，如果dataLen非常地大，则有可能导致内存分配失败，从而导致整个进程崩溃
		if dataLen > configs.Config.Worker.ReceivePackLimit {
			log.Logger.Error().
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Uint32("dataLen", dataLen).
				Uint32("receivePackLimit", configs.Config.Worker.ReceivePackLimit).
				Msg("Worker receive pack size overflow")
			r.ForceClose()
			break
		}
		//获取数据包，这里不必设置读取超时，因为接下来大大概率是有数据的，除非business不按包头包体的协议格式发送
		data := make([]byte, dataLen)
		if _, err = io.ReadAtLeast(connReader, data, len(data)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker receive body failed")
			break
		}
		//business发来心跳
		if bytes.Equal(configs.Config.Worker.HeartbeatMessage, data) {
			continue
		}
		r.receiveCh <- data
	}
}

// LoopCmd 循环处理business发来的各种请求命令
func (r *ConnProcessor) LoopCmd() {
	defer func() {
		if err := recover(); err != nil {
			log.Logger.Error().
				Stack().Err(nil).
				Type("recoverType", err).
				Interface("recover", err).
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker cmd coroutine is closed")
			time.Sleep(5 * time.Second)
			quit.Wg.Add(1)
			go r.LoopCmd()
		} else {
			log.Logger.Debug().
				Int32("events", r.GetEvents()).
				Str("connId", r.connId).
				Msg("Worker cmd coroutine is closed")
		}
		quit.Wg.Done()
	}()
	var cmd uint32
	var callback CmdCallback
	var ok bool
	for data := range r.receiveCh {
		if len(data) > 3 {
			cmd = binary.BigEndian.Uint32(data[0:4])
			if callback, ok = r.cmdCallback[netsvrProtocol.Cmd(cmd)]; ok {
				callback(data[4:], r)
				continue
			}
		}
		log.Logger.Error().
			Int32("events", r.GetEvents()).
			Str("connId", r.connId).
			Uint32("cmd", cmd).
			Hex("dataHex", data).
			Msg("Unknown netsvrProtocol.Cmd")
		//1. business搞错了指令
		//2. business发来的心跳字符串与configs.Config.Worker.HeartbeatMessage配置不一致
		//关掉重来是最好的办法
		r.ForceClose()
		return
	}
}

// RegisterCmd 注册各种命令
func (r *ConnProcessor) RegisterCmd(cmd netsvrProtocol.Cmd, callback CmdCallback) {
	r.cmdCallback[cmd] = callback
}

// GetEvents 返回business进程可以接收的events
func (r *ConnProcessor) GetEvents() int32 {
	return atomic.LoadInt32(&r.events)
}

// SetEvents 设置business进程可以接收的events
func (r *ConnProcessor) SetEvents(id int32) {
	atomic.StoreInt32(&r.events, id)
}
