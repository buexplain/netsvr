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

import "sync"

// StrSlice 字符串切片的多级内存池
type StrSlice struct {
	pools map[int]*sync.Pool
	step  int
	mux   *sync.RWMutex
}

func NewStrSlice(step int) *StrSlice {
	return &StrSlice{
		pools: map[int]*sync.Pool{},
		step:  step,
		mux:   &sync.RWMutex{},
	}
}

func (r *StrSlice) Get(capacity int) *[]string {
	poolIndex := (capacity + r.step - 1) / r.step
	r.mux.RLock()
	pool, ok := r.pools[poolIndex]
	r.mux.RUnlock()
	if ok {
		return pool.Get().(*[]string)
	}
	//这里返回切片的指针，因为在回收环节，往sync.Pool.Put的时候有一个转any的过程
	//如果被转的数据大于runtime.eface的data字段可存储大小，则会调用runtime.convTslice将被转数据转换为unsafe.Pointer，从而导致内存分配
	//@see https://research.swtch.com/interfaces
	s := make([]string, 0, poolIndex*r.step)
	return &s
}

func (r *StrSlice) Put(s *[]string) {
	poolIndex := (cap(*s) + r.step - 1) / r.step
	*s = (*s)[:0]
	//先用读锁看看是否存在pool
	r.mux.RLock()
	pool, ok := r.pools[poolIndex]
	r.mux.RUnlock()
	if ok {
		pool.Put(s)
		return
	}
	//再用互斥锁创建一个新的pool
	r.mux.Lock()
	//拿到锁后再次检查是否已经被其它拿到过锁的协程创建了pool
	pool, ok = r.pools[poolIndex]
	if ok {
		r.mux.Unlock()
		pool.Put(s)
		return
	}
	//从来没被创建过，直接创建一个新的pool
	pool = &sync.Pool{
		New: func() any {
			tmp := make([]string, 0, poolIndex*r.step)
			return &tmp
		},
	}
	r.pools[poolIndex] = pool
	r.mux.Unlock()
	pool.Put(s)
}
