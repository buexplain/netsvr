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

package connProcessor

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/pkg/quit"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	"time"
)

type WorkerCmdCallback func(data []byte, processor *ConnProcessor)
type BusinessCmdCallback func(tf *netsvrProtocol.Transfer, param string, processor *ConnProcessor)

type ConnProcessor struct {
	//business与worker的连接
	conn net.Conn
	//退出信号
	closeCh chan struct{}
	//要发送给连接的数据
	sendCh chan []byte
	//发送缓冲区
	sendBuf     bytes.Buffer
	sendDataLen uint32
	//从连接中读取的数据
	receiveCh chan *netsvrProtocol.Router
	//当前连接的workerId
	workerId int32
	//worker发来的各种命令的回调函数
	workerCmdCallback map[int32]WorkerCmdCallback
	//客户发来的各种命令的回调函数
	businessCmdCallback map[protocol.Cmd]BusinessCmdCallback
	//取消注册成功的信号
	unregisterCancel context.CancelFunc
}

func NewConnProcessor(conn net.Conn, workerId int32) *ConnProcessor {
	return &ConnProcessor{
		conn:                conn,
		closeCh:             make(chan struct{}),
		sendCh:              make(chan []byte, 1000),
		sendBuf:             bytes.Buffer{},
		sendDataLen:         0,
		receiveCh:           make(chan *netsvrProtocol.Router, 1000),
		workerId:            workerId,
		workerCmdCallback:   map[int32]WorkerCmdCallback{},
		businessCmdCallback: map[protocol.Cmd]BusinessCmdCallback{},
	}
}

func (r *ConnProcessor) LoopHeartbeat() {
	t := time.NewTicker(time.Duration(35) * time.Second)
	defer func() {
		if err := recover(); err != nil {
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Msg("Business heartbeat coroutine is closed")
		} else {
			log.Logger.Debug().Msg("Business heartbeat coroutine is closed")
		}
		t.Stop()
	}()
	for {
		select {
		case <-r.closeCh:
			return
		case <-t.C:
			//这个心跳一定要发，否则服务端会把连接干掉
			r.Send(netsvrProtocol.PingMessage)
		}
	}
}

func (r *ConnProcessor) GetCloseCh() <-chan struct{} {
	return r.closeCh
}

// ForceClose 优雅的强制关闭，发给worker的数据会被丢弃，worker发来的数据会被处理
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
		//这里丢弃的数据有可能是让worker发给客户的，也有可能是只给worker的
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
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int32("workerId", r.workerId).Msg("Business send coroutine is closed")
		} else {
			log.Logger.Debug().Int32("workerId", r.workerId).Msg("Business send coroutine is closed")
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
		log.Logger.Error().Err(err).Msg("Business send to worker buffer failed")
		//写缓冲区失败，重置缓冲区
		r.sendBuf.Reset()
		return
	}
	//设置写超时
	if err = r.conn.SetWriteDeadline(time.Now().Add(time.Second * 60)); err != nil {
		r.ForceClose()
		log.Logger.Error().Err(err).Msg("Business SetWriteDeadline to worker conn failed")
		return
	}
	//一次性写入到连接中
	_, err = r.sendBuf.WriteTo(r.conn)
	if err != nil {
		r.ForceClose()
		log.Logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker failed")
		return
	}
	//写入成功，重置缓冲区
	r.sendBuf.Reset()
}

func (r *ConnProcessor) Send(data []byte) {
	defer func() {
		//因为有可能已经阻塞在r.sendCh <- data的时候，收到<-r.producerCh信号
		//然后因为close(r.sendCh)，最终导致send on closed channel
		_ = recover()
	}()
	select {
	case <-r.closeCh:
		//收到关闭信号，不再生产
		return
	default:
		r.sendCh <- data
	}
}

func (r *ConnProcessor) LoopReceive() {
	defer func() {
		//关闭数据管道，不再生产数据进去，让消费者协程退出
		close(r.receiveCh)
		//有可能发起取消注册后，网关突然关闭了连接，这里就可以直接通知r.UnregisterWorker方法退出等待
		r.UnregisterWorkerOk()
		//打印日志信息
		if err := recover(); err != nil {
			quit.Execute("Business receive coroutine error")
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int32("workerId", r.workerId).Msg("Business receive coroutine is closed")
		} else {
			quit.Execute("Worker server shutdown")
			log.Logger.Debug().Int32("workerId", r.workerId).Msg("Business receive coroutine is closed")
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
		//获取前4个字节，确定数据包长度
		if _, err = io.ReadFull(r.conn, dataLenBuf); err != nil {
			//读失败了，直接干掉这个连接，让business重新连接，因为缓冲区的tcp流已经脏了，程序无法拆包
			r.ForceClose()
			break
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		//判断装载数据的缓存区是否足够
		if dataLen > dataBufCap {
			//分配一块更大的，如果dataLen非常的大，则有可能导致内存分配失败
			dataBufCap = dataLen
			dataBuf = make([]byte, dataBufCap)
		} else {
			//清空当前的
			dataBuf = dataBuf[:0]
			dataBuf = dataBuf[0:dataLen]
		}
		//获取数据包
		if _, err = io.ReadAtLeast(r.conn, dataBuf, int(dataLen)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Msg("Business receive failed")
			break
		}
		//worker响应心跳
		if bytes.Equal(netsvrProtocol.PongMessage, dataBuf[0:dataLen]) {
			continue
		}
		router := &netsvrProtocol.Router{}
		if err := proto.Unmarshal(dataBuf[0:dataLen], router); err != nil {
			log.Logger.Error().Err(err).Msg("Proto unmarshal internalProtocol.Router failed")
			continue
		}
		log.Logger.Debug().Stringer("cmd", router.Cmd).Msg("Business receive worker command")
		r.receiveCh <- router
	}
}

// LoopCmd 循环处理worker发来的各种请求命令
func (r *ConnProcessor) LoopCmd() {
	//添加到进程结束时的等待中，这样客户发来的数据都会被处理完毕
	quit.Wg.Add(1)
	defer func() {
		quit.Wg.Done()
		if err := recover(); err != nil {
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int32("workerId", r.workerId).Msg("Business cmd coroutine is closed")
			time.Sleep(5 * time.Second)
			go r.LoopCmd()
		} else {
			log.Logger.Debug().Int32("workerId", r.workerId).Msg("Business cmd coroutine is closed")
		}
	}()
	for data := range r.receiveCh {
		r.cmd(data)
	}
}

func (r *ConnProcessor) cmd(router *netsvrProtocol.Router) {
	if router.Cmd == netsvrProtocol.Cmd_Transfer {
		//解析出worker转发过来的对象
		tf := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(router.Data, tf); err != nil {
			log.Logger.Error().Err(err).Msg("Proto unmarshal internalProtocol.Transfer failed")
			return
		}
		//解析出业务路由对象
		clientRoute := new(protocol.ClientRouter)
		if err := json.Unmarshal(tf.Data, clientRoute); err != nil {
			log.Logger.Debug().Err(err).Msg("Parse protocol.ClientRouter failed")
			return
		}
		log.Logger.Debug().Stringer("cmd", clientRoute.Cmd).Msg("Business receive client command")
		//客户发来的命令
		if callback, ok := r.businessCmdCallback[clientRoute.Cmd]; ok {
			callback(tf, clientRoute.Data, r)
			return
		}
		//客户请求了错误的命令
		log.Logger.Debug().Interface("cmd", clientRoute.Cmd).Msg("Unknown protocol.clientRoute.Cmd")
		return
	}
	//回调worker发来的命令
	if callback, ok := r.workerCmdCallback[int32(router.Cmd)]; ok {
		callback(router.Data, r)
		return
	}
	//worker传递了未知的命令
	log.Logger.Error().Interface("cmd", router.Cmd).Msg("Unknown internalProtocol.Router.Cmd")
}

func (r *ConnProcessor) RegisterWorkerCmd(cmd interface{}, callback WorkerCmdCallback) {
	if c, ok := cmd.(netsvrProtocol.Cmd); ok {
		r.workerCmdCallback[int32(c)] = callback
		return
	}
	if c, ok := cmd.(protocol.Cmd); ok {
		r.workerCmdCallback[int32(c)] = callback
	}
}

func (r *ConnProcessor) RegisterBusinessCmd(cmd protocol.Cmd, callback BusinessCmdCallback) {
	r.businessCmdCallback[cmd] = callback
}

// GetWorkerId 返回workerId
func (r *ConnProcessor) GetWorkerId() int32 {
	return r.workerId
}

func (r *ConnProcessor) RegisterWorker(processCmdGoroutineNum uint32) error {
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_Register
	reg := &netsvrProtocol.Register{}
	reg.Id = r.workerId
	//让worker为我开启n条协程来处理我的请求
	reg.ProcessCmdGoroutineNum = processCmdGoroutineNum
	router.Data, _ = proto.Marshal(reg)
	data, _ := proto.Marshal(router)
	err := binary.Write(r.conn, binary.BigEndian, uint32(len(data)))
	_, err = r.conn.Write(data)
	return err
}

func (r *ConnProcessor) UnregisterWorkerOk() {
	if r.unregisterCancel != nil {
		r.unregisterCancel()
	}
}

// UnregisterWorker 向网关发起取消注册，并等待网关返回取消成功的信息
func (r *ConnProcessor) UnregisterWorker() {
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_Unregister
	pt, _ := proto.Marshal(router)
	ctx, cancel := context.WithCancel(context.Background())
	r.unregisterCancel = cancel
	r.Send(pt)
	//如果网关没有返回数据，这里则会一直阻塞，所以加个倒计时兜底，确保本函数不会被阻塞
	t := time.After(time.Second * 120)
	select {
	case <-t:
		return
	case <-ctx.Done():
		return
	}
}
