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
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v3/netsvr"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/pkg/quit"
	"netsvr/test/business/configs"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	"netsvr/test/pkg/utils/netSvrPool"
	"time"
)

var heartbeatMessage []byte

func init() {
	heartbeatMessage = make([]byte, 4+len(configs.Config.WorkerHeartbeatMessage))
	binary.BigEndian.PutUint32(heartbeatMessage[0:4], uint32(len(configs.Config.WorkerHeartbeatMessage)))
	copy(heartbeatMessage[4:], configs.Config.WorkerHeartbeatMessage)
}

type WorkerCmdCallback func(data []byte, processor *ConnProcessor)
type BusinessCmdCallback func(tf *netsvrProtocol.Transfer, param string, processor *ConnProcessor)

type ConnProcessor struct {
	//business与worker的连接
	conn net.Conn
	//注册后，worker服务器下发id
	connId string
	//退出信号
	closeCh chan struct{}
	//要发送给连接的数据
	sendCh chan []byte
	//从连接中读取的数据
	receiveCh chan []byte
	//当前连接可接收处理的事件
	events int32
	//worker发来的各种命令的回调函数
	workerCmdCallback map[int32]WorkerCmdCallback
	//客户发来的各种命令的回调函数
	businessCmdCallback map[protocol.Cmd]BusinessCmdCallback
}

func NewConnProcessor(conn net.Conn, events int32) *ConnProcessor {
	return &ConnProcessor{
		conn:                conn,
		closeCh:             make(chan struct{}),
		sendCh:              make(chan []byte, 1000),
		receiveCh:           make(chan []byte, 1000),
		workerCmdCallback:   map[int32]WorkerCmdCallback{},
		businessCmdCallback: map[protocol.Cmd]BusinessCmdCallback{},
		events:              events,
	}
}

func (r *ConnProcessor) LoopHeartbeat() {
	t := time.NewTicker(25 * time.Second)
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
			r.sendCh <- heartbeatMessage
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
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int32("events", r.events).Msg("Business send coroutine is closed")
		} else {
			log.Logger.Debug().Int32("events", r.events).Msg("Business send coroutine is closed")
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
		if err = r.conn.SetWriteDeadline(time.Now().Add(60 * time.Second)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Str("connId", r.connId).Int32("events", r.GetEvents()).Msg("Business SetWriteDeadline to worker conn failed")
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
			log.Logger.Error().Err(err).Str("connId", r.connId).Int32("events", r.GetEvents()).Str("businessToWorkerData", base64.StdEncoding.EncodeToString(data)).Msg("Business send to worker failed")
			return
		}
		//写入过部分数据，tcp管道已污染，对端已经无法拆包，必须关闭连接
		r.ForceClose()
		log.Logger.Error().Err(err).Str("connId", r.connId).Int32("events", r.GetEvents()).Msg("Business send to worker failed")
		return
	}
}

func (r *ConnProcessor) Send(message proto.Message, cmd netsvrProtocol.Cmd) {
	select {
	case <-r.closeCh:
		//收到关闭信号，不再生产数据
		return
	default:
		data := make([]byte, 8)
		//先写业务层的cmd
		binary.BigEndian.PutUint32(data[4:8], uint32(cmd))
		if message == nil {
			//再写包头
			binary.BigEndian.PutUint32(data[0:4], 4)
			//发送出去
			r.sendCh <- data
			return
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
		}
	}
}

func (r *ConnProcessor) LoopReceive() {
	defer func() {
		//关闭数据管道，不再生产数据进去，让消费者协程退出
		close(r.receiveCh)
		//打印日志信息
		if err := recover(); err != nil {
			quit.Execute("Business receive coroutine error")
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int32("events", r.events).Msg("Business receive coroutine is closed")
		} else {
			quit.Execute("Worker server shutdown")
			log.Logger.Debug().Int32("events", r.events).Msg("Business receive coroutine is closed")
		}
	}()
	//包头专用
	dataLenBuf := make([]byte, 0, 4)
	//包体专用
	var err error
	connReader := bufio.NewReaderSize(r.conn, 4096)
	for {
		//获取前4个字节，确定数据包长度
		dataLenBuf = dataLenBuf[:4]
		if _, err = io.ReadAtLeast(connReader, dataLenBuf, 4); err != nil {
			//读失败了，直接干掉这个连接，让business重新连接，因为缓冲区的tcp流已经脏了，程序无法拆包
			r.ForceClose()
			break
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		//获取数据包
		data := make([]byte, dataLen)
		if _, err = io.ReadAtLeast(connReader, data, len(data)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Str("connId", r.connId).Int32("events", r.GetEvents()).Msg("Business receive body failed")
			break
		}
		r.receiveCh <- data
	}
}

// LoopCmd 循环处理worker发来的各种请求命令
func (r *ConnProcessor) LoopCmd() {
	defer func() {
		quit.Wg.Done()
		if err := recover(); err != nil {
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int32("events", r.events).Msg("Business cmd coroutine is closed")
			time.Sleep(5 * time.Second)
			//添加到进程结束时的等待中，这样客户发来的数据都会被处理完毕
			quit.Wg.Add(1)
			go r.LoopCmd()
		} else {
			log.Logger.Debug().Int32("events", r.events).Msg("Business cmd coroutine is closed")
		}
	}()
	for data := range r.receiveCh {
		r.cmd(data)
	}
}

func (r *ConnProcessor) cmd(data []byte) {
	if len(data) < 4 {
		return
	}
	var cmd uint32
	cmd = binary.BigEndian.Uint32(data[0:4])
	if netsvrProtocol.Cmd(cmd) == netsvrProtocol.Cmd_Transfer {
		//解析出worker转发过来的对象
		tf := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(data[4:], tf); err != nil {
			log.Logger.Error().Err(err).Str("connId", r.connId).Int32("events", r.GetEvents()).Msg("Proto unmarshal internalProtocol.Transfer failed")
			return
		}
		//如果开始与结尾的字符不是花括号，说明不是有效的json字符串，则把数据原样echo回去
		if l := len(tf.Data); l > 0 && (tf.Data[0] == 123 && tf.Data[l-1] == 125) == false {
			r.echo(tf)
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
	if callback, ok := r.workerCmdCallback[int32(cmd)]; ok {
		callback(data[4:], r)
		return
	}
	//worker传递了未知的命令
	log.Logger.Error().Uint32("cmd", cmd).Msg("Unknown internalProtocol.Router.Cmd")
}

// 将用户发来的数据原样返回给用户
func (r *ConnProcessor) echo(tf *netsvrProtocol.Transfer) {
	sc := netsvrProtocol.SingleCast{}
	sc.UniqId = tf.UniqId
	sc.Data = make([]byte, len(tf.Data))
	copy(sc.Data[3:], tf.Data)
	r.Send(&sc, netsvrProtocol.Cmd_SingleCast)
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

// GetEvents 返回本business进程可处理的是事件
func (r *ConnProcessor) GetEvents() int32 {
	return r.events
}

// RegisterToNetsvrWorker 向网关注册本business进程可处理的事件
func (r *ConnProcessor) RegisterToNetsvrWorker(processCmdGoroutineNum uint32) error {
	data := make([]byte, 8)
	//先写业务层的cmd
	binary.BigEndian.PutUint32(data[4:8], uint32(netsvrProtocol.Cmd_Register))
	var err error
	//再编码业务数据
	pm := proto.MarshalOptions{}
	reg := &netsvrProtocol.RegisterReq{}
	reg.Events = r.events
	//让worker为我开启n条协程来处理我的请求
	reg.ProcessCmdGoroutineNum = processCmdGoroutineNum
	data, err = pm.MarshalAppend(data, reg)
	if err != nil {
		return err
	}
	//再写包头
	binary.BigEndian.PutUint32(data[0:4], uint32(len(data)-4))
	//将数据发送到网关
	var writeLen int
	for {
		writeLen, err = r.conn.Write(data)
		if err != nil {
			return err
		}
		//没有错误，但是只写入部分数据，继续写入
		if writeLen < len(data) {
			data = data[writeLen:]
			continue
		}
		//写入成功
		break
	}
	//发送注册信息成功，开始接收注册结果
	//获取前4个字节，确定数据包长度
	data = make([]byte, 4)
	if _, err = io.ReadFull(r.conn, data); err != nil {
		return err
	}
	//这里采用大端序
	dataLen := binary.BigEndian.Uint32(data)
	data = make([]byte, dataLen)
	//获取数据包
	if _, err = io.ReadAtLeast(r.conn, data, int(dataLen)); err != nil {
		return err
	}
	//解码数据包
	cmd := binary.BigEndian.Uint32(data[0:4])
	if netsvrProtocol.Cmd(cmd) != netsvrProtocol.Cmd_Register {
		return errors.New("expecting the netsvr to return a response to the register cmd")
	}
	payload := netsvrProtocol.RegisterResp{}
	if err = proto.Unmarshal(data[4:], &payload); err != nil {
		return err
	}
	if payload.Code == netsvrProtocol.RegisterRespCode_Success {
		r.connId = payload.ConnId
		return nil
	}
	return errors.New(payload.Message)
}

// UnregisterWorker 向网关发起取消注册，并等待网关返回取消成功的信息
func (r *ConnProcessor) UnregisterWorker() {
	req := &netsvrProtocol.UnRegisterReq{}
	req.ConnId = r.connId
	resp := netsvrProtocol.UnRegisterResp{}
	netSvrPool.Request(req, netsvrProtocol.Cmd_Unregister, &resp)
}
