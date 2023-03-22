/**
* Copyright 2022 buexplain@qq.com
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
	"bytes"
	"encoding/binary"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/configs"
	"netsvr/internal/log"
	"netsvr/pkg/quit"
	"sync/atomic"
	"time"
)

type CmdCallback func(data []byte, processor *ConnProcessor)

type ConnProcessor struct {
	//business与worker的连接
	conn net.Conn
	//退出信号
	closeCh chan struct{}
	//要发送给连接的数据
	sendCh chan []byte
	//发送缓冲区
	sendBuf     *bytes.Buffer
	sendDataLen uint32
	//从连接中读取的数据
	receiveCh chan *netsvrProtocol.Router
	//当前连接的workerId
	workerId int32
	//各种命令的回调函数
	cmdCallback map[netsvrProtocol.Cmd]CmdCallback
}

func NewConnProcessor(conn net.Conn) *ConnProcessor {
	return &ConnProcessor{
		conn:        conn,
		closeCh:     make(chan struct{}),
		sendCh:      make(chan []byte, 1000),
		sendBuf:     &bytes.Buffer{},
		sendDataLen: 0,
		receiveCh:   make(chan *netsvrProtocol.Router, 1000),
		workerId:    0,
		cmdCallback: map[netsvrProtocol.Cmd]CmdCallback{},
	}
}

func (r *ConnProcessor) GetCloseCh() <-chan struct{} {
	return r.closeCh
}

// ForceClose 优雅的强制关闭，发给business的数据会被丢弃，business发来的数据会被处理
func (r *ConnProcessor) ForceClose() {
	defer func() {
		_ = recover()
	}()
	select {
	case <-r.closeCh:
		return
	default:
		//通知所有生产者，不再生产数据
		close(r.closeCh)
		//因为生产者协程(r.sendCh <- data)可能被阻塞，而没有收到关闭信号，所以要丢弃数据，直到所有生产者不再阻塞
		//因为r.sendCh是空的，所以消费者协程可能阻塞，所以要丢弃数据，直到判断出管子是空的，再关闭管子，让消费者协程感知管子已经关闭，可以退出协程
		//这里丢弃的数据有可能是客户发的，也有可能是只给business的
		for {
			select {
			case _, ok := <-r.sendCh:
				if ok {
					continue
				} else {
					time.Sleep(time.Millisecond * 100)
					_ = r.conn.Close()
					return
				}
			default:
				//关闭管子，让消费者协程退出
				close(r.sendCh)
				time.Sleep(time.Millisecond * 100)
				_ = r.conn.Close()
				return
			}
		}
	}
}

func (r *ConnProcessor) LoopSend() {
	defer func() {
		//打印日志信息
		if err := recover(); err != nil {
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int32("workerId", r.GetWorkerId()).Msg("Worker send coroutine is closed")
		} else {
			log.Logger.Debug().Int32("workerId", r.GetWorkerId()).Msg("Worker send coroutine is closed")
		}
	}()
	for data := range r.sendCh {
		select {
		case <-r.closeCh:
			//收到关闭信号
			return
		default:
			r.send(data)
		}
	}
}

func (r *ConnProcessor) send(data []byte) {
	r.sendDataLen = uint32(len(data))
	//先写包头，注意这是大端序
	r.sendBuf.WriteByte(byte(r.sendDataLen >> 24))
	r.sendBuf.WriteByte(byte(r.sendDataLen >> 16))
	r.sendBuf.WriteByte(byte(r.sendDataLen >> 8))
	r.sendBuf.WriteByte(byte(r.sendDataLen))
	//再写包体
	var err error
	if _, err = r.sendBuf.Write(data); err != nil {
		log.Logger.Error().Err(err).Msg("Worker send to business buffer failed")
		//写缓冲区失败，重置缓冲区
		r.sendBuf.Reset()
		return
	}
	//设置写超时
	if err = r.conn.SetWriteDeadline(time.Now().Add(configs.Config.Worker.SendDeadline)); err != nil {
		r.ForceClose()
		log.Logger.Error().Err(err).Msg("Worker SetWriteDeadline to business conn failed")
		return
	}
	//一次性写入到连接中
	_, err = r.sendBuf.WriteTo(r.conn)
	if err != nil {
		//如果失败，则有可能写入了部分字节，进而导致business不能解包
		//而且business没有第一时间处理数据，极有可能是阻塞住了
		//所以强制关闭连接，让数据直接丢弃是最好的选择
		//否则这里的阻塞会蔓延整个网关进程，导致处理客户心跳的协程都没有，最终导致所有客户连接被服务端强制关闭
		//两害相权取其轻
		r.ForceClose()
		log.Logger.Error().Err(err).Int32("workerId", r.GetWorkerId()).Bytes("workerToBusinessData", data).Msg("Worker send to business failed")
		return
	}
	//写入成功，重置缓冲区
	r.sendBuf.Reset()
}

func (r *ConnProcessor) Send(data []byte) {
	select {
	case <-r.closeCh:
		//收到关闭信号，不再生产数据
		return
	default:
		//可能有大量的协程阻塞在这里
		r.sendCh <- data
	}
}

func (r *ConnProcessor) LoopReceive() {
	defer func() {
		//关闭数据管道，不再生产数据进去，让消费者协程退出
		close(r.receiveCh)
		//也许business没有主动发送注销指令，只是关闭了连接，所以这里必须去操作一次注销函数，确保business连接从连接管理器中移除，不再接受的数据转发
		if unregisterWorker, ok := r.cmdCallback[netsvrProtocol.Cmd_Unregister]; ok {
			unregisterWorker(nil, r)
		}
		//打印日志信息
		if err := recover(); err != nil {
			log.Logger.Error().Stack().Err(nil).Type("recoverType", err).Interface("recover", err).Int32("workerId", r.GetWorkerId()).Msg("Worker receive coroutine is closed")
		} else {
			log.Logger.Debug().Int32("workerId", r.GetWorkerId()).Msg("Worker receive coroutine is closed")
		}
	}()
	//包头专用
	dataLenBuf := make([]byte, 4)
	//包体专用
	var dataBufCap uint32 = 0
	var dataBuf []byte
	var err error
	for {
		dataLenBuf = dataLenBuf[:0]
		dataLenBuf = dataLenBuf[0:4]
		//设置读超时时间，再这个时间之内，business没有发数据过来，则会发生超时错误，导致连接被关闭
		if err = r.conn.SetReadDeadline(time.Now().Add(configs.Config.Worker.ReadDeadline)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Msg("Worker SetReadDeadline to business conn failed")
			break
		}
		//获取前4个字节，确定数据包长度
		if _, err = io.ReadFull(r.conn, dataLenBuf); err != nil {
			//读失败了，直接干掉这个连接，让business端重新连接进来，因为缓冲区的tcp流已经脏了，程序无法拆包
			//关掉重来，是最好的办法
			r.ForceClose()
			break
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		//判断装载数据的缓存区是否足够
		if dataLen > dataBufCap {
			//分配一块更大的，如果dataLen非常地大，则有可能导致内存分配失败，从而导致整个进程崩溃
			dataBufCap = dataLen
			dataBuf = make([]byte, dataBufCap)
		} else {
			//清空当前的
			dataBuf = dataBuf[:0]
			dataBuf = dataBuf[0:dataLen]
		}
		//获取数据包，这里不必设置读取超时，因为接下来大大概率是有数据的，除非business不按包头包体的协议格式发送
		if _, err = io.ReadAtLeast(r.conn, dataBuf, int(dataLen)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Msg("Worker receive failed")
			break
		}
		//business发来心跳
		if bytes.Equal(netsvrProtocol.PingMessage, dataBuf[0:dataLen]) {
			//响应business的心跳
			r.Send(netsvrProtocol.PongMessage)
			continue
		}
		router := &netsvrProtocol.Router{}
		if err = proto.Unmarshal(dataBuf[0:dataLen], router); err != nil {
			log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.Router failed")
			continue
		}
		log.Logger.Debug().Stringer("cmd", router.Cmd).Msg("Worker receive business command")
		r.receiveCh <- router
	}
}

// LoopCmd 循环处理business发来的各种请求命令
func (r *ConnProcessor) LoopCmd() {
	//添加到进程结束时的等待中，这样business发来的数据都会被处理完毕
	quit.Wg.Add(1)
	defer func() {
		quit.Wg.Done()
		if err := recover(); err != nil {
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int32("workerId", r.GetWorkerId()).Msg("Worker cmd coroutine is closed")
			time.Sleep(5 * time.Second)
			go r.LoopCmd()
		} else {
			log.Logger.Debug().Int32("workerId", r.GetWorkerId()).Msg("Worker cmd coroutine is closed")
		}
	}()
	for data := range r.receiveCh {
		r.cmd(data)
	}
}

func (r *ConnProcessor) cmd(router *netsvrProtocol.Router) {
	if callback, ok := r.cmdCallback[router.Cmd]; ok {
		callback(router.Data, r)
		return
	}
	//business搞错了指令，直接关闭连接，让business明白，不能瞎传，代码一定要通过测试
	r.ForceClose()
	log.Logger.Error().Interface("cmd", router.Cmd).Msg("Unknown protocol.router.Cmd")
}

// RegisterCmd 注册各种命令
func (r *ConnProcessor) RegisterCmd(cmd netsvrProtocol.Cmd, callback CmdCallback) {
	r.cmdCallback[cmd] = callback
}

// GetWorkerId 返回business的workerId
func (r *ConnProcessor) GetWorkerId() int32 {
	return atomic.LoadInt32(&r.workerId)
}

// SetWorkerId 设置business的workerId
func (r *ConnProcessor) SetWorkerId(id int32) {
	atomic.StoreInt32(&r.workerId, id)
}
