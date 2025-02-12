/**
* Copyright 2024 buexplain@qq.com
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

package netsvrBusiness

import (
	"encoding/binary"
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"google.golang.org/protobuf/proto"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type MainSocket struct {
	eventHandler           EventInterface
	socket                 *Socket
	workerHeartbeatMessage []byte
	events                 netsvrProtocol.Event
	processCmdGoroutineNum int
	connId                 string
	heartbeatInterval      time.Duration
	sendCh                 chan []byte
	connected              *atomic.Bool
	closedCh               chan struct{}
	wait                   *sync.WaitGroup
}

func NewMainSocket(eventHandler EventInterface, socket *Socket, workerHeartbeatMessage []byte, events netsvrProtocol.Event, processCmdGoroutineNum int, heartbeatInterval time.Duration) *MainSocket {
	tmp := &MainSocket{
		eventHandler:           eventHandler,
		socket:                 socket,
		workerHeartbeatMessage: make([]byte, len(workerHeartbeatMessage)),
		processCmdGoroutineNum: processCmdGoroutineNum,
		events:                 events,
		heartbeatInterval:      heartbeatInterval,
		sendCh:                 make(chan []byte, 100),
		connected:              &atomic.Bool{},
		closedCh:               make(chan struct{}),
		wait:                   new(sync.WaitGroup),
	}
	copy(tmp.workerHeartbeatMessage, workerHeartbeatMessage)
	return tmp
}

func (r *MainSocket) GetWorkerAddr() string {
	return r.socket.GetWorkerAddr()
}

func (r *MainSocket) LoopHeartbeat() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("loopHeartbeat panic", "err", err)
			} else {
				logger.Info("loopHeartbeat " + r.GetWorkerAddr() + " quit")
			}
		}()
		t := time.NewTicker(r.heartbeatInterval)
		defer t.Stop()
		for {
			select {
			case <-r.closedCh:
				return
			case <-t.C:
				r.Send(r.workerHeartbeatMessage)
			}
		}
	}()
}

func (r *MainSocket) LoopSend() {
	r.wait.Add(1)
	go func() {
		defer func() {
			r.wait.Done()
			if err := recover(); err != nil {
				logger.Error("loopSend panic", "err", err)
			} else {
				logger.Info("loopSend " + r.GetWorkerAddr() + " quit")
			}
		}()
		for message := range r.sendCh {
			if r.socket.Send(message) == false && r.isConnected() {
				r.reconnect()
			}
		}
	}()
}

func (r *MainSocket) LoopReceive() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("loopReceive panic", "err", err)
			} else {
				logger.Info("loopReceive " + r.GetWorkerAddr() + " quit")
			}
		}()
		for {
			message := r.socket.Receive()
			if message != nil {
				r.processEvent(message)
				continue
			}
			if r.isConnected() == false {
				break
			}
			r.reconnect()
		}
	}()
}

func (r *MainSocket) processEvent(message []byte) {
	r.wait.Add(1)
	go func() {
		defer func() {
			r.wait.Done()
			if err := recover(); err != nil {
				logger.Error("processEvent panic", "err", err, "stack", debug.Stack())
			}
		}()
		cmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(message[0:4]))
		if cmd == netsvrProtocol.Cmd_Transfer {
			transfer := &netsvrProtocol.Transfer{}
			err := proto.Unmarshal(message[4:], transfer)
			if err != nil {
				return
			}
			r.eventHandler.OnMessage(transfer)
			return
		}
		if cmd == netsvrProtocol.Cmd_ConnClose {
			connClose := &netsvrProtocol.ConnClose{}
			err := proto.Unmarshal(message[4:], connClose)
			if err != nil {
				return
			}
			r.eventHandler.OnClose(connClose)
			return
		}
		if cmd == netsvrProtocol.Cmd_ConnOpen {
			connOpen := &netsvrProtocol.ConnOpen{}
			err := proto.Unmarshal(message[4:], connOpen)
			if err != nil {
				return
			}
			r.eventHandler.OnOpen(connOpen)
			return
		}
		logger.Error("unknown cmd", "cmd", cmd)
	}()
}

func (r *MainSocket) isConnected() bool {
	return r.connected.Load()
}

func (r *MainSocket) Connect() bool {
	if r.connected.CompareAndSwap(false, true) {
		if r.socket.Connect() {
			return true
		}
		r.connected.Store(false)
	}
	return false
}

func (r *MainSocket) reconnect() {
	//第一时间重连
	if r.socket.Connect() {
		r.Register()
		return
	}
	//其它协程已经在执行重连操作，休眠3秒，等待重连操作结束
	t := time.NewTimer(time.Second * 3)
	defer t.Stop()
	<-t.C
}

func (r *MainSocket) Send(message []byte) {
	defer func() {
		if err := recover(); err != nil {
			if runtimeError, ok := err.(runtime.Error); ok && runtimeError.Error() == "send on closed channel" {
				//这个错误是无解的，因为正常情况下，channel的关闭是在生产者协程进行的
				//但是现在这里的生产者是多个，并且现在是读取或者是写失败产生的关闭，这里没有关闭的理由
				//所以只能降低日志级别，生产环境无需在意这个日志
				logger.Debug("Send panic", "err", err)
			} else {
				logger.Error("Send panic", "err", err)
			}
		}
	}()
	r.sendCh <- message
}

func (r *MainSocket) Register() bool {
	req := &netsvrProtocol.RegisterReq{}
	req.Events = int32(r.events)
	req.ProcessCmdGoroutineNum = uint32(r.processCmdGoroutineNum)
	message := make([]byte, 4)
	binary.BigEndian.PutUint32(message[0:4], uint32(netsvrProtocol.Cmd_Register))
	var err error
	message, err = (proto.MarshalOptions{}).MarshalAppend(message, req)
	if err != nil {
		logger.Error("marshal netsvrProtocol.RegisterReq failed", "error", err)
		return false
	}
	if r.socket.Send(message) == false {
		return false
	}
	message = r.socket.Receive()
	if message == nil {
		return false
	}
	resp := &netsvrProtocol.RegisterResp{}
	err = proto.Unmarshal(message[4:], resp)
	if err != nil {
		logger.Error("unmarshal netsvrProtocol.RegisterResp failed", "error", err)
		return false
	}
	if resp.Code != 0 {
		logger.Error("register to "+r.GetWorkerAddr()+" failed", "code", resp.Code, "message", resp.Message)
		return false
	}
	r.connId = resp.ConnId
	logger.Info("register to "+r.GetWorkerAddr()+" success", "connId", r.connId)
	return true
}

func (r *MainSocket) Unregister() bool {
	if r.connected.Load() == false {
		return true
	}
	req := &netsvrProtocol.UnRegisterReq{}
	req.ConnId = r.connId
	message := make([]byte, 4)
	binary.BigEndian.PutUint32(message[0:4], uint32(netsvrProtocol.Cmd_Unregister))
	var err error
	message, err = (proto.MarshalOptions{}).MarshalAppend(message, req)
	if err != nil {
		logger.Error("marshal netsvrProtocol.UnRegisterReq failed", "error", err)
		return false
	}
	socket := NewSocket(r.socket.workerAddr, r.socket.receiveTimeout, r.socket.sendTimeout, r.socket.connectTimeout)
	if socket.Connect() == false {
		return false
	}
	defer socket.Close()
	if socket.Send(message) == false {
		return false
	}
	message = socket.Receive()
	if message == nil {
		return false
	}
	resp := &netsvrProtocol.UnRegisterResp{}
	err = proto.Unmarshal(message[4:], resp)
	if err != nil {
		logger.Error("unmarshal netsvrProtocol.UnRegisterResp failed", "error", err)
		return false
	}
	logger.Info("unregister to "+r.GetWorkerAddr()+" success", "connId", r.connId)
	r.connId = ""
	return true
}

func (r *MainSocket) Close() {
	if r.connected.CompareAndSwap(true, false) == false {
		return
	}
	close(r.closedCh)
	close(r.sendCh)
	r.wait.Wait()
	r.socket.Close()
}
