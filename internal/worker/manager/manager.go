package manager

import (
	"math/rand"
	"sync"
	"sync/atomic"
)

type collect struct {
	conn  []*ConnProcessor
	index uint64
	lock  sync.RWMutex
}

func (r *collect) Get() *ConnProcessor {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if len(r.conn) == 0 {
		return nil
	}
	r.index++
	return r.conn[r.index%uint64(len(r.conn))]
}

func (r *collect) Set(conn *ConnProcessor) {
	r.lock.Lock()
	defer r.lock.Unlock()
	exist := false
	for _, v := range r.conn {
		if v == conn {
			exist = true
			break
		}
	}
	if exist == false {
		r.conn = append(r.conn, conn)
	}
}

func (r *collect) Del(conn *ConnProcessor) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for k, v := range r.conn {
		if v == conn {
			r.conn = append(r.conn[0:k], r.conn[k+1:]...)
		}
	}
}

// MaxWorkerId 工作进程的编号范围
const MaxWorkerId = 999
const MinWorkerId = 1

type manager map[int]*collect

func (r manager) Get(workerId int) *ConnProcessor {
	if c, ok := r[workerId]; ok {
		return c.Get()
	}
	return nil
}

func (r manager) Set(workerId int, conn *ConnProcessor) {
	if c, ok := r[workerId]; ok {
		c.Set(conn)
	}
}

func (r manager) Del(workerId int, conn *ConnProcessor) {
	if c, ok := r[workerId]; ok {
		c.Del(conn)
	}
}

// Manager 管理所有的工作进程
var Manager manager

func init() {
	Manager = manager{}
	for i := MinWorkerId; i <= MaxWorkerId; i++ {
		//这里浪费一点内存，全部初始化好，读取的时候就不用动态初始化
		Manager[i] = &collect{conn: []*ConnProcessor{}, index: rand.Uint64(), lock: sync.RWMutex{}}
	}
}

// 处理客户连接关闭的工作进程编号
var processConnCloseWorkerId *int32

func SetProcessConnCloseWorkerId(workerId int32) {
	atomic.StoreInt32(processConnCloseWorkerId, workerId)
}
func GetProcessConnCloseWorkerId() int {
	return int(atomic.LoadInt32(processConnCloseWorkerId))
}
func init() {
	var currentProcessConnCloseWorkerId int32 = 0
	processConnCloseWorkerId = &currentProcessConnCloseWorkerId
}

// 处理客户连接关闭的工作进程编号
var processConnOpenWorkerId *int32

func SetProcessConnOpenWorkerId(workerId int32) {
	atomic.StoreInt32(processConnOpenWorkerId, workerId)
}
func GetProcessConnOpenWorkerId() int {
	return int(atomic.LoadInt32(processConnOpenWorkerId))
}
func init() {
	var currentProcessConnOpenWorkerId int32 = 0
	processConnOpenWorkerId = &currentProcessConnOpenWorkerId
}
