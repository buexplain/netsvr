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

// Package manager 维护客户连接的模块
package manager

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"sync"
)

type collect struct {
	//uniqId --> *websocket.Conn
	conn map[string]*websocket.Conn
	mux  *sync.RWMutex
}

func (r *collect) Len() int {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return len(r.conn)
}

func (r *collect) GetConnections(connections *[]*websocket.Conn) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	for _, conn := range r.conn {
		*connections = append(*connections, conn)
	}
}

func (r *collect) GetUniqIds(uniqIds *[]string) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	for uniqId := range r.conn {
		*uniqIds = append(*uniqIds, uniqId)
	}
}

func (r *collect) Has(uniqId string) bool {
	r.mux.RLock()
	defer r.mux.RUnlock()
	_, ok := r.conn[uniqId]
	return ok
}

func (r *collect) Get(uniqId string) *websocket.Conn {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if c, ok := r.conn[uniqId]; ok {
		return c
	}
	return nil
}

func (r *collect) Set(uniqId string, conn *websocket.Conn) {
	r.mux.Lock()
	r.conn[uniqId] = conn
	r.mux.Unlock()
}

func (r *collect) Del(uniqId string) {
	r.mux.Lock()
	delete(r.conn, uniqId)
	r.mux.Unlock()
}

const managerLen = 8

type manager [managerLen]*collect

func (r manager) index(uniqId string) uint32 {
	//算子常数与标准库保持一致 Go/src/hash/fnv/fnv.go
	var hash uint32 = 2166136261
	for i := 0; i < len(uniqId); i++ {
		hash *= 16777619
		hash ^= uint32(uniqId[i])
	}
	return hash % managerLen
}

func (r manager) Has(uniqId string) bool {
	return r[r.index(uniqId)].Has(uniqId)
}

func (r manager) Get(uniqId string) *websocket.Conn {
	return r[r.index(uniqId)].Get(uniqId)
}

func (r manager) Set(uniqId string, conn *websocket.Conn) {
	r[r.index(uniqId)].Set(uniqId, conn)
}

func (r manager) Del(uniqId string) {
	r[r.index(uniqId)].Del(uniqId)
}

func (r manager) Len() int {
	length := 0
	for _, c := range r {
		length += c.Len()
	}
	return length
}

var Manager manager

func init() {
	for i := 0; i < len(Manager); i++ {
		Manager[i] = &collect{conn: make(map[string]*websocket.Conn, 4096), mux: &sync.RWMutex{}}
	}
}
