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
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v2/netsvr"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/configs"
	"netsvr/internal/log"
	"netsvr/pkg/quit"
	"strconv"
	"sync/atomic"
	"time"
)

var pongMessage []byte
var registerId *uint32

func init() {
	pongMessage = make([]byte, 4+len(netsvrProtocol.PongMessage))
	binary.BigEndian.PutUint32(pongMessage[0:4], uint32(len(netsvrProtocol.PongMessage)))
	copy(pongMessage[4:], netsvrProtocol.PongMessage)
	var registerIdBody uint32 = 0
	registerId = &registerIdBody
}

func getRegisterIdPrefix() string {
	strconv.FormatUint(uint64(atomic.AddUint32(registerId, 1)), 10)
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, atomic.AddUint32(registerId, 1))
	return hex.EncodeToString(tmp)
}

type CmdCallback func(data []byte, processor *ConnProcessor)

type ConnProcessor struct {
	//business与worker的连接
	conn       net.Conn
	registerId string
	//退出信号
	closeCh chan struct{}
	//要发送给连接的数据
	sendCh chan []byte
	//从连接中读取的数据
	receiveCh chan []byte
	//当前连接的workerId
	workerId int32
	//各种命令的回调函数
	cmdCallback map[netsvrProtocol.Cmd]CmdCallback
}

func NewConnProcessor(conn net.Conn) *ConnProcessor {
	return &ConnProcessor{
		conn:        conn,
		registerId:  getRegisterIdPrefix() + conn.RemoteAddr().String(),
		closeCh:     make(chan struct{}),
		sendCh:      make(chan []byte, configs.Config.Worker.SendChanCap),
		receiveCh:   make(chan []byte, 1000),
		workerId:    0,
		cmdCallback: map[netsvrProtocol.Cmd]CmdCallback{},
	}
}

func (r *ConnProcessor) GetRegisterId() string {
	return r.registerId
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
		//从关闭管理器中移除掉自己
		Shutter.Del(r)
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
			log.Logger.Debug().Int32("workerId", r.GetWorkerId()).Str("registerId", r.registerId).Msg("Worker send coroutine is closed")
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
	var err error
	var writeLen int
	totalLen := len(data)
	//写入到连接中
	for {
		//设置写超时
		if err = r.conn.SetWriteDeadline(time.Now().Add(configs.Config.Worker.SendDeadline)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Str("registerId", r.registerId).Int32("workerId", r.GetWorkerId()).Msg("Worker SetWriteDeadline to business conn failed")
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
		//没有写入任何数据，tcp管道未被污染，丢弃本次数据，并打印日志
		if totalLen == len(data[writeLen:]) {
			log.Logger.Error().Err(err).Str("registerId", r.registerId).Int32("workerId", r.GetWorkerId()).Str("workerToBusinessData", base64.StdEncoding.EncodeToString(data)).Msg("Worker send to business failed")
			return
		}
		//写入过部分数据，tcp管道已污染，对端已经无法拆包，必须关闭连接
		r.ForceClose()
		log.Logger.Error().Err(err).Str("registerId", r.registerId).Int32("workerId", r.GetWorkerId()).Msg("Worker send to business failed")
		return
	}
}

func (r *ConnProcessor) Send(message proto.Message, cmd netsvrProtocol.Cmd) int {
	select {
	case <-r.closeCh:
		//收到关闭信号，不再生产数据
		return 0
	default:
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
}

func (r *ConnProcessor) LoopReceive() {
	defer func() {
		//关闭数据管道，不再生产数据进去，让消费者协程退出
		close(r.receiveCh)
		//也许business没有主动发送注销指令，只是关闭了连接，所以这里必须确保business连接从连接管理器中移除，不再接受的数据转发
		Manager.Del(r.workerId, r.registerId)
		//打印日志信息
		if err := recover(); err != nil {
			log.Logger.Error().Stack().Err(nil).Type("recoverType", err).Interface("recover", err).Msg("Worker receive coroutine is closed")
		} else {
			log.Logger.Debug().Int32("workerId", r.GetWorkerId()).Str("registerId", r.registerId).Msg("Worker receive coroutine is closed")
		}
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
			log.Logger.Error().Err(err).Str("registerId", r.registerId).Int32("workerId", r.GetWorkerId()).Msg("Worker SetReadDeadline to business conn failed")
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
			log.Logger.Error().Str("registerId", r.registerId).Int32("workerId", r.GetWorkerId()).Uint32("dataLen", dataLen).Uint32("receivePackLimit", configs.Config.Worker.ReceivePackLimit).Msg("Worker receive pack size overflow")
			r.ForceClose()
			break
		}
		//获取数据包，这里不必设置读取超时，因为接下来大大概率是有数据的，除非business不按包头包体的协议格式发送
		data := make([]byte, dataLen)
		if _, err = io.ReadAtLeast(connReader, data, len(data)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Str("registerId", r.registerId).Int32("workerId", r.GetWorkerId()).Msg("Worker receive body failed")
			break
		}
		//business发来心跳
		if bytes.Equal(netsvrProtocol.PingMessage, data) {
			//响应business的心跳
			r.sendCh <- pongMessage
			continue
		}
		r.receiveCh <- data
	}
}

// LoopCmd 循环处理business发来的各种请求命令
func (r *ConnProcessor) LoopCmd() {
	defer func() {
		quit.Wg.Done()
		if err := recover(); err != nil {
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int32("workerId", r.GetWorkerId()).Msg("Worker cmd coroutine is closed")
			time.Sleep(5 * time.Second)
			//添加到进程结束时的等待中，这样business发来的数据都会被处理完毕
			quit.Wg.Add(1)
			go r.LoopCmd()
		} else {
			log.Logger.Debug().Int32("workerId", r.GetWorkerId()).Str("registerId", r.registerId).Msg("Worker cmd coroutine is closed")
		}
	}()
	for data := range r.receiveCh {
		r.cmd(data)
	}
}

func (r *ConnProcessor) cmd(data []byte) {
	var cmd uint32
	if len(data) > 3 {
		cmd = binary.BigEndian.Uint32(data[0:4])
		if callback, ok := r.cmdCallback[netsvrProtocol.Cmd(cmd)]; ok {
			callback(data[4:], r)
			return
		}
	}
	//business搞错了指令，直接关闭连接，让business明白，不能瞎传，代码一定要通过测试
	r.ForceClose()
	log.Logger.Error().Uint32("cmd", cmd).Int32("workerId", r.GetWorkerId()).Msg("Unknown netsvrProtocol.Cmd")
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
