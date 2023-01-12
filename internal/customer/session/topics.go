package session

import (
	"github.com/RoaringBitmap/roaring"
	"sync"
)

type topics struct {
	data map[string]roaring.Bitmap
	mux  sync.RWMutex
}

func (r *topics) Set(topics []string, sessionId uint32) {
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		bitmap, ok := r.data[topic]
		if !ok {
			bitmap = roaring.Bitmap{}
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
		}
	}
}

func (r *topics) Get(topic string) *roaring.Bitmap {
	r.mux.Lock()
	defer r.mux.Unlock()
	if bitmap, ok := r.data[topic]; ok {
		return bitmap.Clone()
	}
	return nil
}

func (r *topics) Gets(topics []string) *roaring.Bitmap {
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := roaring.Bitmap{}
	for _, topic := range topics {
		if bitmap, ok := r.data[topic]; ok {
			//将所有主题的session id都合并到一个大的bitmap中
			ret.Or(&bitmap)
		}
	}
	return &ret
}

func (r *topics) Count() int {
	r.mux.Lock()
	defer r.mux.Unlock()
	return len(r.data)
}

// Topics 登记主题
var Topics *topics

func init() {
	Topics = &topics{data: map[string]roaring.Bitmap{}, mux: sync.RWMutex{}}
}
