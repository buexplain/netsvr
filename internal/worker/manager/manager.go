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

package manager

import (
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"math/rand"
	"sync"
	"sync/atomic"
)

type collect struct {
	conn  []*ConnProcessor
	index uint32
	mux   *sync.RWMutex
}

func (r *collect) Get() *ConnProcessor {
	index := atomic.AddUint32(&r.index, 1)
	r.mux.RLock()
	defer r.mux.RUnlock()
	if len(r.conn) == 0 {
		return nil
	}
	return r.conn[index%uint32(len(r.conn))]
}

func (r *collect) Set(conn *ConnProcessor) {
	r.mux.Lock()
	defer r.mux.Unlock()
	exist := false
	for _, v := range r.conn {
		if v == conn {
			exist = true
			break
		}
	}
	if exist == false {
		r.conn = append(r.conn, conn)
	}
}

func (r *collect) Del(connId string) bool {
	if connId == "" {
		return false
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	for k, v := range r.conn {
		if v.GetConnId() == connId {
			r.conn = append(r.conn[0:k], r.conn[k+1:]...)
			return true
		}
	}
	return false
}

type manager map[netsvrProtocol.Event]*collect

func (r manager) Get(event netsvrProtocol.Event) *ConnProcessor {
	return r[event].Get()
}

func (r manager) Set(conn *ConnProcessor) {
	events := conn.GetEvents()
	for _, v := range netsvrProtocol.Event_value {
		if netsvrProtocol.Event(v) == netsvrProtocol.Event_Placeholder {
			continue
		}
		if events&v == v {
			r[netsvrProtocol.Event(v)].Set(conn)
		}
	}
}

func (r manager) Del(connId string) bool {
	ret := false
	for _, v := range netsvrProtocol.Event_value {
		if netsvrProtocol.Event(v) == netsvrProtocol.Event_Placeholder {
			continue
		}
		if r[netsvrProtocol.Event(v)].Del(connId) {
			ret = true
		}
	}
	return ret
}

// Manager 管理所有的business连接
var Manager manager

func init() {
	Manager = make(manager)
	for _, v := range netsvrProtocol.Event_value {
		if netsvrProtocol.Event(v) == netsvrProtocol.Event_Placeholder {
			continue
		}
		Manager[netsvrProtocol.Event(v)] = &collect{conn: []*ConnProcessor{}, index: rand.Uint32(), mux: &sync.RWMutex{}}
	}
}
