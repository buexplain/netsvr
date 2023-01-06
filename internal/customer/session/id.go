package session

import (
	"github.com/RoaringBitmap/roaring"
	"github.com/buexplain/netsvr/configs"
	"github.com/lesismal/nbio/logging"
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
	lock sync.Mutex
}

// Get 分配一个session id出去
func (r *id) Get() uint32 {
	r.lock.Lock()
	for {
		r.inc++
		if r.inc > r.max {
			r.inc = r.min
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
		allocated: roaring.Bitmap{},
		lock:      sync.Mutex{},
	}
	//单机范围五千万，uint32总共可以分配255台机器
	//客户端根据session id的大小可以推算出该session id所在的网关机器
	var step uint32 = 5000 * 10000
	Id.max = step * uint32(configs.Config.NetServerId)
	Id.min = Id.max - step + 1
	Id.inc = Id.min - 1
	logging.Info("Session id range of the current net server %d is %d ~ %d", configs.Config.NetServerId, Id.min, Id.max)
}
