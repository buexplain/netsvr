/**
* Copyright 2022 buexplain@qq.com
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
	"math/rand"
	"netsvr/pkg/constant"
	"sync"
	"sync/atomic"
)

type collect struct {
	conn  []*ConnProcessor
	index uint64
	mux   *sync.RWMutex
}

func (r *collect) Get() *ConnProcessor {
	index := atomic.AddUint64(&r.index, 1)
	r.mux.RLock()
	defer r.mux.RUnlock()
	if len(r.conn) == 0 {
		return nil
	}
	return r.conn[index%uint64(len(r.conn))]
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

func (r *collect) Del(conn *ConnProcessor) {
	r.mux.Lock()
	defer r.mux.Unlock()
	for k, v := range r.conn {
		if v == conn {
			r.conn = append(r.conn[0:k], r.conn[k+1:]...)
		}
	}
}

type manager [constant.MaxWorkerId + 1]*collect

func (r manager) Get(workerId int) *ConnProcessor {
	if workerId < constant.MinWorkerId || workerId > constant.MaxWorkerId {
		return nil
	}
	return r[workerId].Get()
}

func (r manager) Set(workerId int, conn *ConnProcessor) {
	if workerId >= constant.MinWorkerId && workerId <= constant.MaxWorkerId {
		r[workerId].Set(conn)
	}
}

func (r manager) Del(workerId int, conn *ConnProcessor) {
	if workerId >= constant.MinWorkerId && workerId <= constant.MaxWorkerId {
		r[workerId].Del(conn)
	}
}

// Manager 管理所有的business连接
var Manager manager

func init() {
	Manager = manager{}
	for i := constant.MinWorkerId; i <= constant.MaxWorkerId; i++ {
		//这里浪费一点内存，全部初始化好，读取的时候就不用动态初始化
		Manager[i] = &collect{conn: []*ConnProcessor{}, index: rand.Uint64(), mux: &sync.RWMutex{}}
	}
}
