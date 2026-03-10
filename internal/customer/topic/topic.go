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
	"netsvr/internal/utils/slicePool"
	"sync"
	"unsafe"
)

// shardCount 分段锁数量，必须是 2 的幂次方以便用位运算取模
const shardCount = 256
const shardMask = shardCount - 1 // 255 = 0b11111111

type shard struct {
	mux  sync.RWMutex
	data map[string]map[string]struct{} // topic → set(uniqId)
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

// CountAll 统计每个主题的人数
func (r *collect) CountAll() (ret map[string]int32) {
	// 预估算容量
	capacity := 0
	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		capacity += len(sd.data)
		sd.mux.RUnlock()
	}
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

// Count 统计某几个主题的人数
func (r *collect) Count(topics []string) (ret map[string]int32) {
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

// SetBySlice 设置主题与 uniqId 的对应关系
func (r *collect) SetBySlice(topics []string, uniqId string) {
	if len(topics) == 0 || uniqId == "" {
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
				c[uniqId] = struct{}{}
			} else {
				c = make(map[string]struct{}, 8) // 预分配小容量
				c[uniqId] = struct{}{}
				sd.data[topic] = c
			}
		}
		sd.mux.Unlock()
	}
}

// PullAndReturnUniqIds 删除某几个主题，并返回主题包含的 uniqId
// ⚠️ 所有权说明：
//   - 返回的 map[string]struct{} 是原内部 map 的直接引用
//   - 调用方获得独占所有权，可安全修改/遍历/丢弃
//   - 该 topic 从管理器中完全移除，后续 SetBySlice 会创建新 map
func (r *collect) PullAndReturnUniqIds(topics []string) (ret map[string]map[string]struct{}) {
	if len(topics) == 0 {
		return nil
	}
	ret = make(map[string]map[string]struct{}, len(topics))
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

// GetUniqIds 获取某个主题的所有 uniqId（使用外部 slicePool 复用内存）
// 调用方使用完后必须调用 sp.Put(uniqIds) 归还到池
func (r *collect) GetUniqIds(topic string, sp *slicePool.StrSlice) *[]string {
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
	uniqIds := sp.Get(len(c))
	for uniqId := range c {
		*uniqIds = append(*uniqIds, uniqId)
	}
	return uniqIds
}

// GetUniqIdsByTopics 获取多个主题的所有 uniqId
// ⚠️ 注意：此方法仍会分配 map 和 slice，高频调用建议改用迭代器模式
func (r *collect) GetUniqIdsByTopics(topics []string) (ret map[string][]string) {
	if len(topics) == 0 {
		return nil
	}
	ret = make(map[string][]string, len(topics))
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
				// ⚠️ 此处仍会分配，如需优化可传入 slicePool
				uniqIds := make([]string, 0, len(c))
				for uniqId := range c {
					uniqIds = append(uniqIds, uniqId)
				}
				ret[topic] = uniqIds
			}
		}
		sd.mux.RUnlock()
	}
	return ret
}

// Get 获取所有的主题
func (r *collect) Get() []string {
	// 预估算容量
	capacity := 0
	for i := range r.shards {
		sd := &r.shards[i]
		sd.mux.RLock()
		capacity += len(sd.data)
		sd.mux.RUnlock()
	}
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

// DelByMap 删除主题与 uniqId 的对应关系
func (r *collect) DelByMap(topics map[string]struct{}, uniqId string) {
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
				delete(c, uniqId)
				if len(c) == 0 {
					delete(sd.data, topic)
				}
			}
		}
		sd.mux.Unlock()
	}
}

// DelBySlice 删除主题与 uniqId 的对应关系
func (r *collect) DelBySlice(topics []string, uniqId string) {
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
				delete(c, uniqId)
				if len(c) == 0 {
					delete(sd.data, topic)
				}
			}
		}
		sd.mux.Unlock()
	}
}

// Topic 全局实例
var Topic *collect

func init() {
	Topic = &collect{}
	for i := range Topic.shards {
		Topic.shards[i].data = make(map[string]map[string]struct{}, 64)
	}
}
