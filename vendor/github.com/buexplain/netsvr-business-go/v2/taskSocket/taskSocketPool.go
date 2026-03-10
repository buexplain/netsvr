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

package taskSocket

import (
	"github.com/buexplain/netsvr-business-go/v2/log"
	"time"
)

type Pool struct {
	pool              chan *TaskSocket
	size              chan struct{}
	factory           *Factory
	waitTimeout       time.Duration
	heartbeatInterval time.Duration
	heartbeatMessage  []byte
	closedCh          chan struct{}
}

func NewPool(size int, factory *Factory, waitTimeout time.Duration, heartbeatInterval time.Duration, heartbeatMessage []byte) *Pool {
	tmp := &Pool{}
	tmp.factory = factory
	tmp.waitTimeout = waitTimeout
	tmp.heartbeatInterval = heartbeatInterval
	tmp.heartbeatMessage = make([]byte, len(heartbeatMessage))
	copy(tmp.heartbeatMessage, heartbeatMessage)
	tmp.closedCh = make(chan struct{})
	tmp.pool = make(chan *TaskSocket, size)
	tmp.size = make(chan struct{}, size)
	for i := 0; i < size; i++ {
		tmp.size <- struct{}{}
	}
	return tmp
}

func (t *Pool) GetAddr() string {
	return t.factory.GetAddr()
}

func (t *Pool) Get() *TaskSocket {
	if len(t.pool) == 0 {
		select {
		case <-t.size:
			socket := t.factory.Make(t)
			if socket == nil || socket.IsConnected() == false {
				log.Error("taskSocketPool " + t.factory.GetAddr() + " new socket failed")
				t.size <- struct{}{}
				return nil
			} else {
				log.Info("taskSocketPool " + t.factory.GetAddr() + " new socket success")
				return socket
			}
		default:
			goto wait
		}
	}
wait:
	if t.waitTimeout == 0 {
		return <-t.pool
	}
	timeout := time.NewTimer(t.waitTimeout)
	defer timeout.Stop()
	select {
	case socket := <-t.pool:
		return socket
	case <-timeout.C:
		log.Error("Pool pool exhausted. Cannot establish new connection before wait_timeout.")
		return nil
	}
}

func (t *Pool) release(socket *TaskSocket) {
	if socket == nil || !socket.IsConnected() {
		t.size <- struct{}{}
		return
	}
	t.pool <- socket
}

func (t *Pool) heartbeat() {
	for i := len(t.pool); i > 0; i-- {
		select {
		case <-t.closedCh:
			return
		case socket := <-t.pool:
			if socket.IsConnected() && socket.Send(t.heartbeatMessage) {
				t.pool <- socket
			} else {
				socket.Close()
				log.Info("taskSocketPool heartbeat " + t.GetAddr() + " socket closed")
				t.release(nil)
			}
		default:
			continue
		}
	}
}

func (t *Pool) LoopHeartbeat() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Error("taskSocketPool loopHeartbeat panic", "err", err)
			} else {
				log.Info("taskSocketPool loopHeartbeat " + t.GetAddr() + " quit")
			}
		}()
		ticker := time.NewTicker(t.heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-t.closedCh:
				return
			case <-ticker.C:
				t.heartbeat()
			}
		}
	}()
}

func (t *Pool) Close() {
	select {
	case <-t.closedCh:
		return
	default:
		close(t.closedCh)
		for i := 0; i < cap(t.size); i++ {
			select {
			case socket := <-t.pool:
				socket.Close()
				t.release(nil)
			default:
				continue
			}
		}
	}
}
