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

// Package topic 维护客户的订阅模块
package topic

import (
	"github.com/panjf2000/gnet/v2"
	"netsvr/internal/utils/slicePool"
	"sync"
	"unsafe"
)

// shardCount 分段锁数量，必须是 2 的幂次方以便用位运算取模
const shardCount = 256
const shardMask = shardCount - 1 // 255 = 0b11111111

type shard struct {
	mux  sync.RWMutex
	data map[string]map[int]gnet.Conn // topic → set(gnet.Conn)
}

type collect struct {
	shards [shardCount]shard
}

func hashTopic(topic string) int {
	//算子常数与标准库保持一致 Go/src/hash/fnv/fnv.go
	data := unsafe.Slice(unsafe.StringData(topic), len(topic))
	var hash uint32 = 2166136261
	for _, c := range data {
		hash *= 16777619
		hash ^= uint32(c)
	}
	return int(hash & uint32(shardMask))
}

// SetRelation 设置多个主题与 conn 的对应关系
func (r *collect) SetRelation(topics []string, conn gnet.Conn) {
	if len(topics) == 0 || conn == nil {
		return
	}
	// 按 shard 分组，减少锁切换次数
	var shardGroups [shardCount][]string
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		idx := hashTopic(topic)
		shardGroups[idx] = append(shardGroups[idx], topic)
	}
	for idx, tList := range shardGroups {
		if len(tList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.Lock()
		for _, topic := range tList {
			c, ok := sd.data[topic]
			if ok {
				c[conn.Fd()] = conn
			} else {
				c = make(map[int]gnet.Conn, 8) // 预分配小容量
				c[conn.Fd()] = conn
				sd.data[topic] = c
			}
		}
		sd.mux.Unlock()
	}
}

// DelRelationByMap 删除多个主题与 conn 的对应关系
func (r *collect) DelRelationByMap(topics map[string]struct{}, conn gnet.Conn) {
	if len(topics) == 0 {
		return
	}
	// 按 shard 分组，减少锁切换次数
	var shardGroups [shardCount][]string
	for topic := range topics {
		if topic == "" {
			continue
		}
		idx := hashTopic(topic)
		shardGroups[idx] = append(shardGroups[idx], topic)
	}
	for idx, tList := range shardGroups {
		if len(tList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.Lock()
		for _, topic := range tList {
			c, ok := sd.data[topic]
			if ok {
				delete(c, conn.Fd())
				if len(c) == 0 {
					delete(sd.data, topic)
				}
			}
		}
		sd.mux.Unlock()
	}
}

// DelRelationBySlice 删除多个主题与 conn 的对应关系
func (r *collect) DelRelationBySlice(topics []string, conn gnet.Conn) {
	if len(topics) == 0 {
		return
	}
	// 按 shard 分组，减少锁切换次数
	var shardGroups [shardCount][]string
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		idx := hashTopic(topic)
		shardGroups[idx] = append(shardGroups[idx], topic)
	}
	for idx, tList := range shardGroups {
		if len(tList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.Lock()
		for _, topic := range tList {
			c, ok := sd.data[topic]
			if ok {
				delete(c, conn.Fd())
				if len(c) == 0 {
					delete(sd.data, topic)
				}
			}
		}
		sd.mux.Unlock()
	}
}

// Del 删除多个主题，并返回被删除主题包含的 conn
// ⚠️ 所有权说明：
//   - 返回的 map[string]map[int]gnet.Conn 是原内部 map 的直接引用
//   - 调用方获得独占所有权，可安全修改/遍历/丢弃
//   - 该 topic 从管理器中完全移除，后续 SetRelation 会创建新 map
func (r *collect) Del(topics []string) (ret map[string]map[int]gnet.Conn) {
	if len(topics) == 0 {
		return nil
	}
	ret = make(map[string]map[int]gnet.Conn, len(topics))
	// 按 shard 分组
	var shardGroups [shardCount][]string
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		idx := hashTopic(topic)
		shardGroups[idx] = append(shardGroups[idx], topic)
	}
	for idx, tList := range shardGroups {
		if len(tList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.Lock()
		for _, topic := range tList {
			c, ok := sd.data[topic]
			if ok {
				delete(sd.data, topic) //从 shard 移除引用
				ret[topic] = c         // 返回内部 map 的引用
			}
		}
		sd.mux.Unlock()
	}
	return ret
}

// Len 统计主题个数（近似值，跨 shard 累加）
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

// Get 获取全部主题
func (r *collect) Get() []string {
	// 预估算容量
	capacity := r.Len()
	extra := capacity / 10 // 额外 10% 缓冲
	if extra < 256 {
		extra = 256
	} else if extra > 4096 {
		extra = 4096
	}
	topics := make([]string, 0, capacity+extra)
	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		for topic := range sd.data {
			topics = append(topics, topic)
		}
		sd.mux.RUnlock()
	}
	return topics
}

// CountConn 统计全部主题的人数
func (r *collect) CountConn() (ret map[string]int32) {
	// 预估算容量
	capacity := r.Len()
	extra := capacity / 10 // 额外 10% 缓冲
	if extra < 256 {
		extra = 256
	} else if extra > 4096 {
		extra = 4096
	}
	ret = make(map[string]int32, capacity+extra)
	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		for topic, c := range sd.data {
			ret[topic] = int32(len(c))
		}
		sd.mux.RUnlock()
	}
	return ret
}

// CountConnByTopic 统计多个主题的连接数
func (r *collect) CountConnByTopic(topics []string) (ret map[string]int32) {
	if len(topics) == 0 {
		return
	}
	// 按 shard 分组，减少锁切换次数
	var shardGroups [shardCount][]string
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		idx := hashTopic(topic)
		shardGroups[idx] = append(shardGroups[idx], topic)
	}
	ret = make(map[string]int32, len(topics))
	for idx, tList := range shardGroups {
		if len(tList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.RLock()
		for _, topic := range tList {
			c, ok := sd.data[topic]
			if ok {
				ret[topic] = int32(len(c))
			}
		}
		sd.mux.RUnlock()
	}
	return ret
}

// GetConnListByTopic 获取某个主题的所有 conn（使用外部 slicePool 复用内存）
// 调用方使用完后必须调用 sp.Put(connList) 归还到池
func (r *collect) GetConnListByTopic(topic string, sp *slicePool.WsConn) *[]gnet.Conn {
	if topic == "" {
		return nil
	}
	idx := hashTopic(topic)
	sd := &r.shards[idx]
	sd.mux.RLock()
	defer sd.mux.RUnlock()
	c, ok := sd.data[topic]
	if !ok {
		return nil
	}
	// 从 pool 获取 slice，避免分配
	connList := sp.Get(len(c))
	for _, conn := range c {
		*connList = append(*connList, conn)
	}
	return connList
}

// GetConnListByTopics 获取多个主题的所有 conn
func (r *collect) GetConnListByTopics(topics []string) (ret map[string][]gnet.Conn) {
	if len(topics) == 0 {
		return nil
	}
	ret = make(map[string][]gnet.Conn, len(topics))
	// 按 shard 分组，减少锁切换次数
	var shardGroups [shardCount][]string
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		idx := hashTopic(topic)
		shardGroups[idx] = append(shardGroups[idx], topic)
	}
	for idx, tList := range shardGroups {
		if len(tList) == 0 {
			continue
		}
		sd := &r.shards[idx]
		sd.mux.RLock()
		for _, topic := range tList {
			c, ok := sd.data[topic]
			if ok {
				connList := make([]gnet.Conn, 0, len(c))
				for _, conn := range c {
					connList = append(connList, conn)
				}
				ret[topic] = connList
			}
		}
		sd.mux.RUnlock()
	}
	return ret
}

// GetConnList 获取全部主题的所有 conn
func (r *collect) GetConnList() (ret map[string][]gnet.Conn) {
	ret = make(map[string][]gnet.Conn, r.Len())

	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		for topic, c := range sd.data {
			connList := make([]gnet.Conn, 0, len(c))
			for _, conn := range c {
				connList = append(connList, conn)
			}
			ret[topic] = connList
		}
		sd.mux.RUnlock()
	}
	return ret
}

// Topic 全局实例
var Topic *collect

func init() {
	Topic = &collect{}
	for i := range Topic.shards {
		Topic.shards[i].data = make(map[string]map[int]gnet.Conn, 64)
	}
}
