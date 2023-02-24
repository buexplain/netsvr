package topic

import (
	"sync"
)

type collect struct {
	//topic --> []uniqId
	topics map[string]map[string]struct{}
	mux    sync.RWMutex
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

// Set 设置主题与uniqId的对应关系
func (r *collect) Set(topics []string, uniqId string) {
	if len(topics) == 0 {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		c, ok := r.topics[topic]
		if !ok {
			c = map[string]struct{}{}
			r.topics[topic] = c
		}
		c[uniqId] = struct{}{}
	}
}

// Pull 删除某几个主题
func (r *collect) Pull(topics []string) map[string]map[string]struct{} {
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
func (r *collect) GetUniqIds(topic string) (uniqIds []string) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	c, ok := r.topics[topic]
	if !ok {
		return nil
	}
	uniqIds = make([]string, 0, len(c))
	for uniqId := range c {
		uniqIds = append(uniqIds, uniqId)
	}
	return uniqIds
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

// Del 删除主题与uniqId的对应关系
func (r *collect) Del(topics []string, uniqId string) {
	if len(topics) == 0 {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		c, ok := r.topics[topic]
		if !ok {
			continue
		}
		delete(c, uniqId)
		if len(c) == 0 {
			delete(r.topics, topic)
		}
	}
}

var Topic *collect

func init() {
	Topic = &collect{topics: map[string]map[string]struct{}{}, mux: sync.RWMutex{}}
}
