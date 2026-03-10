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

package mainSocket

import (
	"encoding/binary"
	"github.com/buexplain/netsvr-business-go/v2/contract"
	"github.com/buexplain/netsvr-business-go/v2/log"
	"github.com/buexplain/netsvr-business-go/v2/socket"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"google.golang.org/protobuf/proto"
	"runtime/debug"
	"sync"
	"time"
)

type MainSocket struct {
	eventHandler      contract.EventInterface
	socket            *socket.Socket
	heartbeatMessage  []byte
	events            netsvrProtocol.Event
	connId            string
	heartbeatInterval time.Duration
	closedCh          chan struct{}
	wg                sync.WaitGroup
}

func New(eventHandler contract.EventInterface, socket *socket.Socket, heartbeatMessage []byte, events netsvrProtocol.Event, heartbeatInterval time.Duration) *MainSocket {
	tmp := &MainSocket{
		eventHandler:      eventHandler,
		socket:            socket,
		heartbeatMessage:  make([]byte, len(heartbeatMessage)),
		events:            events,
		heartbeatInterval: heartbeatInterval,
		closedCh:          make(chan struct{}, 1),
		wg:                sync.WaitGroup{},
	}
	copy(tmp.heartbeatMessage, heartbeatMessage)
	return tmp
}

func (r *MainSocket) GetAddr() string {
	return r.socket.GetAddr()
}

func (r *MainSocket) LoopHeartbeat() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Error("mainSocket loopHeartbeat panic", "err", err)
			} else {
				log.Info("mainSocket loopHeartbeat " + r.GetAddr() + " quit")
			}
		}()
		t := time.NewTicker(r.heartbeatInterval)
		defer t.Stop()
		for {
			select {
			case s, ok := <-r.closedCh:
				if ok != false {
					r.closedCh <- s
				}
				return
			case <-t.C:
				if r.socket.IsConnected() {
					r.socket.Send(r.heartbeatMessage)
				}
			}
		}
	}()
}

func (r *MainSocket) LoopReceive() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Error("mainSocket loopReceive panic", "err", err)
			} else {
				log.Info("mainSocket loopReceive " + r.GetAddr() + " quit")
			}
		}()
		for {
			message := r.socket.Receive()
			if message == nil {
				//重连
				if r.socket.Connect() {
					r.Register()
				} else {
					time.Sleep(time.Second * 3)
				}
				continue
			}
			cmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(message[0:4]))
			if cmd == netsvrProtocol.Cmd_Unregister {
				r.closedCh <- struct{}{}
				return
			}
			r.processEvent(cmd, message[4:])
		}
	}()
}

func (r *MainSocket) processEvent(cmd netsvrProtocol.Cmd, message []byte) {
	r.wg.Add(1)
	go func() {
		defer func() {
			r.wg.Done()
			if err := recover(); err != nil {
				log.Error("processEvent panic", "err", err, "stack", debug.Stack())
			}
		}()
		if cmd == netsvrProtocol.Cmd_Transfer {
			transfer := &netsvrProtocol.Transfer{}
			err := proto.Unmarshal(message, transfer)
			if err != nil {
				return
			}
			r.eventHandler.OnMessage(transfer)
			return
		}
		if cmd == netsvrProtocol.Cmd_ConnClose {
			connClose := &netsvrProtocol.ConnClose{}
			err := proto.Unmarshal(message, connClose)
			if err != nil {
				return
			}
			r.eventHandler.OnClose(connClose)
			return
		}
		if cmd == netsvrProtocol.Cmd_ConnOpen {
			connOpen := &netsvrProtocol.ConnOpen{}
			err := proto.Unmarshal(message, connOpen)
			if err != nil {
				return
			}
			r.eventHandler.OnOpen(connOpen)
			return
		}
		log.Error("unknown cmd", "cmd", cmd)
	}()
}

func (r *MainSocket) Connect() bool {
	return r.socket.Connect()
}

func (r *MainSocket) Register() bool {
	req := &netsvrProtocol.RegisterReq{}
	req.Events = int32(r.events)
	message := make([]byte, 4)
	binary.BigEndian.PutUint32(message[0:4], uint32(netsvrProtocol.Cmd_Register))
	var err error
	message, err = (proto.MarshalOptions{}).MarshalAppend(message, req)
	if err != nil {
		log.Error("marshal netsvrProtocol.RegisterReq failed", "error", err)
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
		log.Error("unmarshal netsvrProtocol.RegisterResp failed", "error", err)
		return false
	}
	if resp.Code != 0 {
		log.Error("register to "+r.GetAddr()+" failed", "code", resp.Code, "message", resp.Message)
		return false
	}
	r.connId = resp.ConnId
	log.Info("register to "+r.GetAddr()+" success", "connId", r.connId)
	return true
}

func (r *MainSocket) Unregister() bool {
	//通知心跳协程，退出心跳机制，避免同时写socket
	r.closedCh <- struct{}{}
	<-r.closedCh
	req := &netsvrProtocol.UnRegisterReq{}
	req.ConnId = r.connId
	message := make([]byte, 4)
	binary.BigEndian.PutUint32(message[0:4], uint32(netsvrProtocol.Cmd_Unregister))
	var err error
	message, err = (proto.MarshalOptions{}).MarshalAppend(message, req)
	if err != nil {
		log.Error("marshal netsvrProtocol.UnRegisterReq failed", "error", err)
		return false
	}
	if r.socket.Send(message) {
		//等待socket收到响应
		<-r.closedCh
		log.Info("unregister from "+r.GetAddr()+" success", "connId", r.connId)
		return true
	}
	return false
}

func (r *MainSocket) Close() {
	close(r.closedCh)
	r.wg.Wait()
	r.socket.Close()
}
