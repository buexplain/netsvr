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
	"time"
)

type TaskSocketPool struct {
	pool                   chan *TaskSocket
	size                   chan struct{}
	factory                *TaskSocketFactory
	waitTimeout            time.Duration
	heartbeatInterval      time.Duration
	workerHeartbeatMessage []byte
	closedCh               chan struct{}
}

func NewTaskSocketPool(size int, factory *TaskSocketFactory, waitTimeout time.Duration, heartbeatInterval time.Duration, workerHeartbeatMessage []byte) *TaskSocketPool {
	tmp := &TaskSocketPool{}
	tmp.factory = factory
	tmp.waitTimeout = waitTimeout
	tmp.heartbeatInterval = heartbeatInterval
	tmp.workerHeartbeatMessage = make([]byte, len(workerHeartbeatMessage))
	copy(tmp.workerHeartbeatMessage, workerHeartbeatMessage)
	tmp.closedCh = make(chan struct{})
	tmp.pool = make(chan *TaskSocket, size)
	tmp.size = make(chan struct{}, size)
	for i := 0; i < size; i++ {
		tmp.size <- struct{}{}
	}
	return tmp
}

func (t *TaskSocketPool) GetWorkerAddr() string {
	return t.factory.GetWorkerAddr()
}

func (t *TaskSocketPool) Get() *TaskSocket {
	if len(t.pool) == 0 {
		select {
		case <-t.size:
			socket := t.factory.Make(t)
			if socket == nil || socket.IsConnected() == false {
				t.size <- struct{}{}
			} else {
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
	select {
	case socket := <-t.pool:
		timeout.Stop()
		return socket
	case <-timeout.C:
		timeout.Stop()
		logger.Error("TaskSocketPool pool exhausted. Cannot establish new connection before wait_timeout.")
		return nil
	}
}

func (t *TaskSocketPool) release(taskSocket *TaskSocket) {
	if taskSocket == nil || !taskSocket.IsConnected() {
		t.size <- struct{}{}
		return
	}
	t.pool <- taskSocket
}

func (t *TaskSocketPool) heartbeat() {
	for i := 0; i < cap(t.size); i++ {
		select {
		case <-t.closedCh:
			return
		case socket := <-t.pool:
			if socket.IsConnected() && socket.Send(t.workerHeartbeatMessage) {
				t.pool <- socket
			} else {
				socket.Close()
				t.release(nil)
			}
		default:
			continue
		}
	}
}

func (t *TaskSocketPool) LoopHeartbeat() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("loopHeartbeat panic", "err", err)
			} else {
				logger.Info("loopHeartbeat " + t.GetWorkerAddr() + " quit")
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

func (t *TaskSocketPool) Close() {
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
