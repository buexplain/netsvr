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
	"slices"
	"sync"
	"unsafe"
)

// shardCount 分段锁数量，必须是 2 的幂次方以便用位运算取模
const shardCount = 256
const shardMask = shardCount - 1 // 255 = 0b11111111

type shard struct {
	mux  sync.RWMutex
	data map[string][]string //customerId --> slice(uniqId)
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

// GetCustomerIds 获取所有的customerId
func (r *collect) GetCustomerIds() (customerIds []string) {
	// 预估算容量
	capacity := 0
	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		capacity += len(sd.data)
		sd.mux.RUnlock()
	}
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

// GetUniqIdsByCustomerId 根据customerId获取所有的uniqId
func (r *collect) GetUniqIdsByCustomerId(customerId string) (uniqIds []string) {
	if customerId == "" {
		return nil
	}
	idx := hashCustomerId(customerId)
	sd := &r.shards[idx]
	sd.mux.RLock()
	defer sd.mux.RUnlock()
	if c, ok := sd.data[customerId]; ok {
		uniqIds = make([]string, 0, len(c))
		uniqIds = append(uniqIds, c...)
		return uniqIds
	}
	return nil
}

// GetUniqIdsByCustomerIds 根据customerIds获取所有的uniqId
func (r *collect) GetUniqIdsByCustomerIds(customerIds []string) (customerIdUniqIds map[string][]string) {
	// 按 shard 分组，减少锁切换次数
	var shardGroups [shardCount][]string
	for _, customerId := range customerIds {
		if customerId == "" {
			continue
		}
		idx := hashCustomerId(customerId)
		shardGroups[idx] = append(shardGroups[idx], customerId)
	}
	customerIdUniqIds = make(map[string][]string, len(customerIds))
	for idx, cList := range shardGroups {
		if len(cList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.RLock()
		for _, customerId := range cList {
			if c, ok := sd.data[customerId]; ok {
				uniqIds := make([]string, 0, len(c))
				uniqIds = append(uniqIds, c...)
				customerIdUniqIds[customerId] = uniqIds
			}
		}
		sd.mux.RUnlock()
	}
	return customerIdUniqIds
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

func (r *collect) Set(customerId string, uniqId string) {
	idx := hashCustomerId(customerId)
	sd := &r.shards[idx]
	sd.mux.Lock()
	defer sd.mux.Unlock()
	if c, ok := sd.data[customerId]; ok {
		if slices.Index(c, uniqId) == -1 {
			sd.data[customerId] = append(slices.Grow(c, 1), uniqId)
		}
	} else {
		sd.data[customerId] = []string{uniqId}
	}
}

func (r *collect) Del(customerId string, uniqId string) {
	idx := hashCustomerId(customerId)
	sd := &r.shards[idx]
	sd.mux.Lock()
	defer sd.mux.Unlock()
	if c, ok := sd.data[customerId]; ok {
		index := slices.Index(c, uniqId)
		if index != -1 {
			c = slices.Delete(c, index, index+1)
			if len(c) == 0 {
				delete(sd.data, customerId)
			} else {
				//缩容：如果容量 > 2*长度，重新分配
				if cap(c) > len(c)*2 {
					newC := make([]string, len(c))
					copy(newC, c)
					c = newC
				}
				sd.data[customerId] = c
			}
		}
	}
}

// Binder 全局实例
var Binder *collect

func init() {
	Binder = &collect{}
	for i := range Binder.shards {
		Binder.shards[i].data = make(map[string][]string, 128)
	}
}
