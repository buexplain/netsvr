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
	"netsvr/internal/customer/info"
	"netsvr/internal/utils/slicePool"
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
	data map[string]*wsServer.Codec //uniqId --> *wsServer.Codec
}

type collect struct {
	shards [shardCount]shard
}

func hashUniqId(uniqId string) int {
	//算子常数与标准库保持一致 Go/src/hash/fnv/fnv.go
	data := unsafe.Slice(unsafe.StringData(uniqId), len(uniqId))
	var hash uint32 = 2166136261
	for _, c := range data {
		hash *= 16777619
		hash ^= uint32(c)
	}
	return int(hash & uint32(shardMask))
}

func (r *collect) Has(uniqId string) bool {
	if uniqId == "" {
		return false
	}
	idx := hashUniqId(uniqId)
	sd := &r.shards[idx]
	sd.mux.RLock()
	defer sd.mux.RUnlock()
	_, ok := sd.data[uniqId]
	return ok
}

func (r *collect) Get(uniqId string) *wsServer.Codec {
	if uniqId == "" {
		return nil
	}
	idx := hashUniqId(uniqId)
	sd := &r.shards[idx]
	sd.mux.RLock()
	defer sd.mux.RUnlock()
	if c, ok := sd.data[uniqId]; ok {
		return c
	}
	return nil
}

func (r *collect) Set(uniqId string, conn *wsServer.Codec) {
	idx := hashUniqId(uniqId)
	sd := &r.shards[idx]
	sd.mux.Lock()
	defer sd.mux.Unlock()
	sd.data[uniqId] = conn
}

func (r *collect) Del(uniqId string) {
	if uniqId == "" {
		return
	}
	idx := hashUniqId(uniqId)
	sd := &r.shards[idx]
	sd.mux.Lock()
	defer sd.mux.Unlock()
	delete(sd.data, uniqId)
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

// GetConnections 获取所有连接对象
// ⚠️ 使用说明：
//   - 返回 nil 表示无连接，调用方无需调用 sp.Put()
//   - 返回非 nil 时，调用方使用完后必须调用 sp.Put(connections) 归还
func (r *collect) GetConnections(sp *slicePool.WsConn) (connections *[]*wsServer.Codec) {
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
	connections = sp.Get(capacity + extra)
	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		for _, conn := range sd.data {
			*connections = append(*connections, conn)
		}
		sd.mux.RUnlock()
	}
	return connections
}

func (r *collect) GetCustomerIds(uniqIds []string) (customerIds []string) {
	if len(uniqIds) == 0 {
		return nil
	}
	// 按 shard 分组，减少锁切换次数
	var shardGroups [shardCount][]string
	for _, uniqId := range uniqIds {
		if uniqId == "" {
			continue
		}
		idx := hashUniqId(uniqId)
		shardGroups[idx] = append(shardGroups[idx], uniqId)
	}
	//再取出所有的连接对象，避免嵌套获取锁
	connList := make([]*wsServer.Codec, 0, len(uniqIds))
	for idx, uList := range shardGroups {
		if len(uList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.RLock()
		for _, uniqId := range uList {
			conn, ok := sd.data[uniqId]
			if ok {
				connList = append(connList, conn)
			}
		}
		sd.mux.RUnlock()
	}
	//收集所有连接对应的customerId（使用 map 去重）
	customerIdSet := make(map[string]struct{}, len(connList))
	for _, conn := range connList {
		session, _ := conn.GetSession().(*info.Info)
		customerId := session.GetCustomerIdOnSafe()
		if customerId != "" {
			customerIdSet[customerId] = struct{}{}
		}
	}
	// 转换为 slice
	if len(customerIdSet) == 0 {
		return nil
	}
	customerIds = make([]string, 0, len(customerIdSet))
	for id := range customerIdSet {
		customerIds = append(customerIds, id)
	}
	return customerIds
}

func (r *collect) CountCustomerIds(uniqIds []string) int32 {
	if len(uniqIds) == 0 {
		return 0
	}
	// 按 shard 分组，减少锁切换次数
	var shardGroups [shardCount][]string
	for _, uniqId := range uniqIds {
		if uniqId == "" {
			continue
		}
		idx := hashUniqId(uniqId)
		shardGroups[idx] = append(shardGroups[idx], uniqId)
	}
	//再取出所有的连接对象，避免嵌套获取锁
	connList := make([]*wsServer.Codec, 0, len(uniqIds))
	for idx, uList := range shardGroups {
		if len(uList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.RLock()
		for _, uniqId := range uList {
			conn, ok := sd.data[uniqId]
			if ok {
				connList = append(connList, conn)
			}
		}
		sd.mux.RUnlock()
	}
	//收集所有连接对应的customerId（使用 map 去重计数）
	customerIdSet := make(map[string]struct{}, len(connList))
	for _, conn := range connList {
		session, _ := conn.GetSession().(*info.Info)
		customerId := session.GetCustomerIdOnSafe()
		if customerId != "" {
			customerIdSet[customerId] = struct{}{}
		}
	}
	return int32(len(customerIdSet))
}

func (r *collect) GetUniqIds() (uniqIds []string) {
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
	uniqIds = make([]string, 0, capacity+extra)
	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		for uniqId := range sd.data {
			uniqIds = append(uniqIds, uniqId)
		}
		sd.mux.RUnlock()
	}
	return uniqIds
}

// Manager 全局实例
var Manager *collect

func init() {
	Manager = &collect{}
	for i := range Manager.shards {
		Manager.shards[i].data = make(map[string]*wsServer.Codec, 64*runtime.NumCPU())
	}
}
