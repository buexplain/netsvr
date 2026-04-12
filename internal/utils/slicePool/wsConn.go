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

// Package slicePool 多级内存池
package slicePool

import (
	"netsvr/internal/wsServer"
	"sync"
)

// WsConn int切片的多级内存池
type WsConn struct {
	pools map[int]*sync.Pool
	step  int
	mux   *sync.RWMutex
}

func NewWsConn(step int) *WsConn {
	if step <= 0 {
		step = 16 // 默认步长
	}
	return &WsConn{
		pools: map[int]*sync.Pool{},
		step:  step,
		mux:   &sync.RWMutex{},
	}
}

func (r *WsConn) Get(capacity int) *[]*wsServer.Codec {
	if capacity <= 0 {
		capacity = 1 // 保证最小容量，避免创建空切片
	}
	poolIndex := (capacity + r.step - 1) / r.step
	r.mux.RLock()
	pool, ok := r.pools[poolIndex]
	r.mux.RUnlock()
	if ok {
		return pool.Get().(*[]*wsServer.Codec)
	}
	s := make([]*wsServer.Codec, 0, poolIndex*r.step)
	return &s
}

func (r *WsConn) Put(s *[]*wsServer.Codec) {
	poolIndex := (cap(*s) + r.step - 1) / r.step
	*s = (*s)[:0]
	r.mux.RLock()
	pool, ok := r.pools[poolIndex]
	r.mux.RUnlock()
	if ok {
		pool.Put(s)
		return
	}
	r.mux.Lock()
	pool, ok = r.pools[poolIndex]
	if ok {
		r.mux.Unlock()
		pool.Put(s)
		return
	}
	pool = &sync.Pool{
		New: func() any {
			tmp := make([]*wsServer.Codec, 0, poolIndex*r.step)
			return &tmp
		},
	}
	r.pools[poolIndex] = pool
	r.mux.Unlock()
	pool.Put(s)
}
