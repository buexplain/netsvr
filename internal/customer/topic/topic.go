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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"netsvr/internal/utils/slicePool"
	"sync"
)

type collect struct {
	//topic --> []uniqId
	topics map[string]map[string]struct{}
	mux    *sync.RWMutex
}

// Len 统计主题个数
func (r *collect) Len() int {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return len(r.topics)
}

// CountAll 统计每个主题的人数
func (r *collect) CountAll(items map[string]int32) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	for topic, c := range r.topics {
		items[topic] = int32(len(c))
	}
}

// Count 统计某几个主题的人数
func (r *collect) Count(topics []string, items map[string]int32) {
	if len(topics) == 0 {
		return
	}
	r.mux.RLock()
	defer r.mux.RUnlock()
	for _, topic := range topics {
		c, ok := r.topics[topic]
		if !ok {
			continue
		}
		items[topic] = int32(len(c))
	}
}

// SetBySlice 设置主题与uniqId的对应关系
func (r *collect) SetBySlice(topics []string, uniqId string) {
	if len(topics) == 0 || uniqId == "" {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		c, ok := r.topics[topic]
		if ok {
			if _, ok = c[uniqId]; !ok {
				c[uniqId] = struct{}{}
			}
		} else {
			c = map[string]struct{}{}
			r.topics[topic] = c
			c[uniqId] = struct{}{}
		}
	}
}

// PullAndReturnUniqIds 删除某几个主题，并返回主题包含的uniqId
func (r *collect) PullAndReturnUniqIds(topics []string) map[string]map[string]struct{} {
	if len(topics) == 0 {
		return nil
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := make(map[string]map[string]struct{}, len(topics))
	for _, topic := range topics {
		c, ok := r.topics[topic]
		if !ok {
			continue
		}
		delete(r.topics, topic)
		ret[topic] = c
	}
	return ret
}

// GetUniqIds 获取某个主题的所有uniqId
func (r *collect) GetUniqIds(topic string, slicePool *slicePool.StrSlice) *[]string {
	r.mux.RLock()
	defer r.mux.RUnlock()
	c, ok := r.topics[topic]
	if !ok {
		return nil
	}
	uniqIds := slicePool.Get(len(c))
	for uniqId := range c {
		*uniqIds = append(*uniqIds, uniqId)
	}
	return uniqIds
}

func (r *collect) GetToProtocolTopicUniqIdListResp(topicUniqIdListResp *netsvrProtocol.TopicUniqIdListResp, topics []string) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	for _, topic := range topics {
		c, ok := r.topics[topic]
		if !ok {
			continue
		}
		item := &netsvrProtocol.TopicUniqIdListRespItem{}
		item.UniqIds = make([]string, 0, len(c))
		for uniqId := range c {
			item.UniqIds = append(item.UniqIds, uniqId)
		}
		topicUniqIdListResp.Items[topic] = item
	}
}

// Get 获取所有的主题
func (r *collect) Get() (topics []string) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	topics = make([]string, 0, len(r.topics))
	for topic := range r.topics {
		topics = append(topics, topic)
	}
	return topics
}

// DelByMap 删除主题与uniqId的对应关系
func (r *collect) DelByMap(topics map[string]struct{}, currentUniqId string, previousUniqId string) {
	if len(topics) == 0 {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	if currentUniqId == previousUniqId || previousUniqId == "" {
		for topic := range topics {
			c, ok := r.topics[topic]
			if ok {
				delete(c, currentUniqId)
				if len(c) == 0 {
					delete(r.topics, topic)
				}
			}
		}
		return
	}
	for topic := range topics {
		c, ok := r.topics[topic]
		if ok {
			delete(c, currentUniqId)
			delete(c, previousUniqId)
			if len(c) == 0 {
				delete(r.topics, topic)
			}
		}
	}
}

// DelBySlice 删除主题与uniqId的对应关系
func (r *collect) DelBySlice(topics []string, currentUniqId string, previousUniqId string) {
	if len(topics) == 0 {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	if previousUniqId == "" || currentUniqId == previousUniqId {
		for _, topic := range topics {
			c, ok := r.topics[topic]
			if ok {
				delete(c, currentUniqId)
				if len(c) == 0 {
					delete(r.topics, topic)
				}
			}
		}
		return
	}
	for _, topic := range topics {
		c, ok := r.topics[topic]
		if ok {
			delete(c, currentUniqId)
			delete(c, previousUniqId)
			if len(c) == 0 {
				delete(r.topics, topic)
			}
		}
	}
}

var Topic *collect

func init() {
	Topic = &collect{topics: map[string]map[string]struct{}{}, mux: &sync.RWMutex{}}
}
