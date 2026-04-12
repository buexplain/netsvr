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

// Package binder 网关唯一id与客户业务系统唯一id的绑定关系
package binder

import (
	"netsvr/internal/wsServer"
	"runtime"
	"sync"
	"unsafe"
)

// shardCount 分段锁数量，必须是 2 的幂次方以便用位运算取模
const shardCount = 256
const shardMask = shardCount - 1 // 255 = 0b11111111

type shard struct {
	mux  sync.RWMutex
	data map[string]map[int]*wsServer.Codec //customerId --> set(*wsServer.Codec)
}

type collect struct {
	shards [shardCount]shard
}

func hashCustomerId(customerId string) int {
	//算子常数与标准库保持一致 Go/src/hash/fnv/fnv.go
	data := unsafe.Slice(unsafe.StringData(customerId), len(customerId))
	var hash uint32 = 2166136261
	for _, c := range data {
		hash *= 16777619
		hash ^= uint32(c)
	}
	return int(hash & uint32(shardMask))
}

func (r *collect) SetRelation(customerId string, wsCodec *wsServer.Codec) {
	idx := hashCustomerId(customerId)
	sd := &r.shards[idx]
	sd.mux.Lock()
	defer sd.mux.Unlock()
	c, ok := sd.data[customerId]
	if !ok {
		c = make(map[int]*wsServer.Codec, 1)
		sd.data[customerId] = c
	}
	c[wsCodec.Fd()] = wsCodec
}

func (r *collect) DelRelation(customerId string, wsCodec *wsServer.Codec) {
	idx := hashCustomerId(customerId)
	sd := &r.shards[idx]
	sd.mux.Lock()
	defer sd.mux.Unlock()
	if c, ok := sd.data[customerId]; ok {
		delete(c, wsCodec.Fd())
		if len(c) == 0 {
			delete(sd.data, customerId)
		}
	}
}

func (r *collect) Len() int {
	total := 0
	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		total += len(sd.data)
		sd.mux.RUnlock()
	}
	return total
}

// GetCustomerIds 获取所有的customerId
func (r *collect) GetCustomerIds() (customerIds []string) {
	// 预估算容量
	capacity := r.Len()
	if capacity == 0 {
		return nil
	}
	extra := capacity / 10 // 额外 10% 缓冲
	if extra < 256 {
		extra = 256
	} else if extra > 4096 {
		extra = 4096
	}
	customerIds = make([]string, 0, capacity+extra)
	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		for customerId := range sd.data {
			customerIds = append(customerIds, customerId)
		}
		sd.mux.RUnlock()
	}
	return customerIds
}

// GetConnListByCustomerId 获取某个customerId的conn列表
func (r *collect) GetConnListByCustomerId(customerId string) (connList []*wsServer.Codec) {
	if customerId == "" {
		return nil
	}
	idx := hashCustomerId(customerId)
	sd := &r.shards[idx]
	sd.mux.RLock()
	defer sd.mux.RUnlock()
	if c, ok := sd.data[customerId]; ok {
		connList = make([]*wsServer.Codec, 0, len(c))
		for _, conn := range c {
			connList = append(connList, conn)
		}
		return connList
	}
	return nil
}

// GetConnListByCustomerIds 获取多个customerId的conn列表
func (r *collect) GetConnListByCustomerIds(customerIds []string) (customerIdConnList map[string][]*wsServer.Codec) {
	// 按 shard 分组，减少锁切换次数
	var shardGroups [shardCount][]string
	for _, customerId := range customerIds {
		if customerId == "" {
			continue
		}
		idx := hashCustomerId(customerId)
		shardGroups[idx] = append(shardGroups[idx], customerId)
	}
	customerIdConnList = make(map[string][]*wsServer.Codec, len(customerIds))
	for idx, cList := range shardGroups {
		if len(cList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.RLock()
		for _, customerId := range cList {
			if c, ok := sd.data[customerId]; ok {
				connList := make([]*wsServer.Codec, 0, len(c))
				for _, conn := range c {
					connList = append(connList, conn)
				}
				customerIdConnList[customerId] = connList
			}
		}
		sd.mux.RUnlock()
	}
	return customerIdConnList
}

// Binder 全局实例
var Binder *collect

func init() {
	Binder = &collect{}
	for i := range Binder.shards {
		Binder.shards[i].data = make(map[string]map[int]*wsServer.Codec, 64*runtime.NumCPU())
	}
}
