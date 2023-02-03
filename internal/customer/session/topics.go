package session

import (
	"github.com/RoaringBitmap/roaring"
	"sync"
)

type topics struct {
	data map[string]*roaring.Bitmap
	mux  sync.RWMutex
}

func (r *topics) Set(topics []string, sessionId uint32) {
	if len(topics) == 0 {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		bitmap, ok := r.data[topic]
		if !ok {
			bitmap = &roaring.Bitmap{}
			r.data[topic] = bitmap
		}
		bitmap.Add(sessionId)
	}
}

func (r *topics) Del(topics []string, sessionId uint32) {
	if len(topics) == 0 {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		if bitmap, ok := r.data[topic]; ok {
			bitmap.Remove(sessionId)
			//如果是空的，则连主题一起删除掉
			if bitmap.IsEmpty() {
				delete(r.data, topic)
				bitmap = nil
			}
		}
	}
}

// Get 获取主题的bitmap数据
func (r *topics) Get(topic string) *roaring.Bitmap {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if bitmap, ok := r.data[topic]; ok {
		return bitmap.Clone()
	}
	return nil
}

// Gets 获取主题的bitmap数据
func (r *topics) Gets(topics []string, ret map[string]string) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	for _, topic := range topics {
		if bitmap, ok := r.data[topic]; ok {
			ret[topic], _ = bitmap.ToBase64()
		}
	}
}

// CountConn 统计主题里面的连接数
func (r *topics) CountConn(topics []string, getAll bool, ret map[string]int32) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	//获取全部主题的连接数
	if getAll {
		for topic, bitmap := range r.data {
			ret[topic] = int32(bitmap.GetCardinality())
		}
		return
	}
	//获取某几个主题的连接数
	for _, topic := range topics {
		if bitmap, ok := r.data[topic]; ok {
			ret[topic] = int32(bitmap.GetCardinality())
		}
	}
}

// Count 获取主题数量
func (r *topics) Count() int {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return len(r.data)
}

// Topics 登记主题
var Topics *topics

func init() {
	Topics = &topics{data: map[string]*roaring.Bitmap{}, mux: sync.RWMutex{}}
}
