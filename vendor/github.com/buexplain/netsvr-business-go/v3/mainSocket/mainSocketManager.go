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
	"github.com/buexplain/netsvr-business-go/v3/contract"
	"sync/atomic"
)

// Manager 长连接管理器，统一管理到各网关的 MainSocket
type Manager struct {
	pool      map[string]*MainSocket
	connected atomic.Bool
}

// NewManager 创建一个长连接管理器
func NewManager() *Manager {
	return &Manager{
		pool:      make(map[string]*MainSocket),
		connected: atomic.Bool{},
	}
}

// AddSocket 按长连接的网关地址注册它
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

// Start 连接并注册所有长连接，然后启动收发与心跳；任一步失败会回滚已建立的连接并返回 false。
// 重复调用是幂等的（已启动时直接返回 true）
func (m *Manager) Start() bool {
	if m.connected.CompareAndSwap(false, true) == false {
		return true
	}
	m.connected.Store(m.connect() && m.register())
	return m.connected.Load()
}

// Close 注销并关闭所有长连接
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
