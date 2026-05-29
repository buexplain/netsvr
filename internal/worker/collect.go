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

package worker

import (
	"sync"
	"sync/atomic"
)

type collect struct {
	conn  []*Conn
	index uint32
	mux   sync.RWMutex
}

func (r *collect) Get() *Conn {
	index := atomic.AddUint32(&r.index, 1)
	r.mux.RLock()
	defer r.mux.RUnlock()
	if len(r.conn) == 0 {
		return nil
	}
	return r.conn[index%uint32(len(r.conn))]
}

func (r *collect) Set(conn *Conn) {
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
