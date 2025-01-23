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
	"sync/atomic"
)

type MainSocketManager struct {
	pool      map[string]*MainSocket
	connected *atomic.Bool
}

func NewMainSocketManager() *MainSocketManager {
	return &MainSocketManager{
		pool:      make(map[string]*MainSocket),
		connected: new(atomic.Bool),
	}
}

func (m *MainSocketManager) GetSockets() []*MainSocket {
	if m.connected.Load() {
		sockets := make([]*MainSocket, len(m.pool))
		for _, socket := range m.pool {
			sockets = append(sockets, socket)
		}
		return sockets
	}
	return nil
}

func (m *MainSocketManager) GetSocket(workerAddrAsHex string) *MainSocket {
	if m.connected.Load() {
		socket, ok := m.pool[workerAddrAsHex]
		if ok {
			return socket
		}
	}
	return nil
}

func (m *MainSocketManager) AddSocket(socket *MainSocket) {
	m.pool[WorkerAddrConvertToHex(socket.GetWorkerAddr())] = socket
}

func (m *MainSocketManager) connect() bool {
	completed := make([]*MainSocket, len(m.pool))
	ok := true
	for _, socket := range m.pool {
		if socket.Connect() {
			completed = append(completed, socket)
		} else {
			ok = false
			break
		}
	}
	if ok == false {
		for _, socket := range completed {
			socket.Close()
		}
		return false
	}
	return true
}

func (m *MainSocketManager) register() bool {
	completed := make([]*MainSocket, len(m.pool))
	ok := true
	for _, socket := range m.pool {
		if socket.Register() {
			socket.LoopSend()
			socket.LoopReceive()
			socket.LoopHeartbeat()
			completed = append(completed, socket)
		} else {
			ok = false
			break
		}
	}
	if ok == false {
		for _, socket := range completed {
			socket.Close()
		}
		return false
	}
	return true
}

func (m *MainSocketManager) Start() bool {
	if m.connected.CompareAndSwap(false, true) == false {
		return true
	}
	m.connected.Store(m.connect() && m.register())
	return m.connected.Load()
}

func (m *MainSocketManager) Close() {
	if m.connected.CompareAndSwap(true, false) == false {
		return
	}
	for _, socket := range m.pool {
		socket.Unregister()
	}
	for _, socket := range m.pool {
		socket.Close()
	}
}
