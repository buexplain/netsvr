package session

import (
	"github.com/RoaringBitmap/roaring"
	"sync"
)

type id struct {
	//当前自增的id
	inc uint32
	//已经分配的id集合
	allocated roaring.Bitmap
	//互斥锁
	lock sync.RWMutex
}

// Get 分配一个session id出去
func (r *id) Get() uint32 {
	r.lock.Lock()
	for {
		r.inc++
		if r.inc == 0 {
			continue
		}
		if !r.allocated.Contains(r.inc) {
			r.allocated.Add(r.inc)
			r.lock.Unlock()
			return r.inc
		}
	}
}

// Put 归还一个session id 回来
func (r *id) Put(id uint32) {
	r.lock.Lock()
	r.allocated.Remove(id)
	r.lock.Unlock()
}

// GetAllocated 获取已分配的id集合
func (r *id) GetAllocated() *roaring.Bitmap {
	r.lock.Lock()
	tmp := r.allocated.Clone()
	r.lock.Unlock()
	return tmp
}

var Id *id

func init() {
	Id = &id{
		inc:       0,
		allocated: roaring.Bitmap{},
		lock:      sync.RWMutex{},
	}
}
