package topic

import (
	"sync"
)

type collect struct {
	topic map[string]map[string]struct{}
	mux   sync.RWMutex
}

func (r *collect) Len() int {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return len(r.topic)
}

func (r *collect) CountAll(items map[string]int32) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	for topic, c := range r.topic {
		items[topic] = int32(len(c))
	}
}

func (r *collect) Count(topics []string, items map[string]int32) {
	if len(topics) == 0 {
		return
	}
	r.mux.RLock()
	defer r.mux.RUnlock()
	for _, topic := range topics {
		c, ok := r.topic[topic]
		if !ok {
			continue
		}
		items[topic] = int32(len(c))
	}
}

func (r *collect) Set(topics []string, uniqId string) {
	if len(topics) == 0 {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		c, ok := r.topic[topic]
		if !ok {
			c = map[string]struct{}{}
			r.topic[topic] = c
		}
		c[uniqId] = struct{}{}
	}
}

func (r *collect) Pull(topic string, uniqIds *[]string) *[]string {
	r.mux.Lock()
	defer r.mux.Unlock()
	c, ok := r.topic[topic]
	if !ok {
		return uniqIds
	}
	delete(r.topic, topic)
	if uniqIds == nil {
		tmp := make([]string, 0, len(c))
		uniqIds = &tmp
	}
	for uniqId := range c {
		*uniqIds = append(*uniqIds, uniqId)
	}
	return uniqIds
}

func (r *collect) Get(topic string) (uniqIds []string) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	c, ok := r.topic[topic]
	if !ok {
		return nil
	}
	uniqIds = make([]string, 0, len(c))
	for uniqId := range c {
		uniqIds = append(uniqIds, uniqId)
	}
	return uniqIds
}

func (r *collect) Del(topics []string, uniqId string) {
	if len(topics) == 0 {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		c, ok := r.topic[topic]
		if !ok {
			continue
		}
		delete(c, uniqId)
		if len(c) == 0 {
			delete(r.topic, topic)
		}
	}
}

var Topic *collect

func init() {
	Topic = &collect{topic: map[string]map[string]struct{}{}, mux: sync.RWMutex{}}
}
