package session

import (
	"github.com/RoaringBitmap/roaring"
	"github.com/lesismal/nbio/logging"
	"netsvr/configs"
	"sync"
)

type id struct {
	//当前自增的id
	inc uint32
	//最小值
	min uint32
	//最大值
	max uint32
	//已经分配的id集合
	allocated roaring.Bitmap
	//互斥锁
	lock sync.RWMutex
}

// Pull 分配一个session id出去
func (r *id) Pull() uint32 {
	r.lock.Lock()
	defer r.lock.Unlock()
	for {
		r.inc++
		if r.inc > r.max {
			r.inc = r.min
		}
		if !r.allocated.Contains(r.inc) {
			r.allocated.Add(r.inc)
			return r.inc
		}
	}
}

// Put 归还一个session id 回来
func (r *id) Put(id uint32) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.allocated.Remove(id)
}

// GetAllocated 获取已分配的id集合
func (r *id) GetAllocated() *roaring.Bitmap {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.allocated.Clone()
}

// CountAllocated 获取已分配的id集合大小
func (r *id) CountAllocated() uint64 {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.allocated.GetCardinality()
}

var Id *id

func init() {
	Id = &id{
		allocated: roaring.Bitmap{},
		lock:      sync.RWMutex{},
	}
	Id.min = configs.Config.SessionIdMin
	Id.max = configs.Config.SessionIdMax
	Id.inc = Id.min - 1
	logging.Info("Session id range: %d ~ %d", Id.min, Id.max)
}
