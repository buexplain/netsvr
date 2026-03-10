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
	"github.com/buexplain/netsvr-business-go/v2/contract"
	"sync/atomic"
)

type Manager struct {
	pool      map[string]*MainSocket
	connected atomic.Bool
}

func NewManager() *Manager {
	return &Manager{
		pool:      make(map[string]*MainSocket),
		connected: atomic.Bool{},
	}
}

func (m *Manager) AddSocket(socket *MainSocket) {
	m.pool[contract.AddrConvertToHex(socket.GetAddr())] = socket
}

func (m *Manager) connect() bool {
	completed := make([]*MainSocket, 0, len(m.pool))
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

func (m *Manager) register() bool {
	completed := make([]*MainSocket, 0, len(m.pool))
	ok := true
	for _, socket := range m.pool {
		if socket.Register() {
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

func (m *Manager) Start() bool {
	if m.connected.CompareAndSwap(false, true) == false {
		return true
	}
	m.connected.Store(m.connect() && m.register())
	return m.connected.Load()
}

func (m *Manager) Close() {
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
