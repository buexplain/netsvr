package manager

import (
	"sync"
	"sync/atomic"
)

type collect struct {
	conn  []*connection
	index int
	lock  sync.Mutex
}

func (r *collect) Get() *connection {
	r.lock.Lock()
	defer r.lock.Unlock()
	if len(r.conn) == 0 {
		return nil
	}
	r.index++
	if r.index >= len(r.conn) {
		r.index = 0
	}
	return r.conn[r.index]
}

func (r *collect) Set(conn *connection) {
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

func (r *collect) Del(conn *connection) {
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

func (r manager) Get(workerId int) *connection {
	if c, ok := r[workerId]; ok {
		return c.Get()
	}
	return nil
}

func (r manager) Set(workerId int, conn *connection) {
	if c, ok := r[workerId]; ok {
		c.Set(conn)
	}
}

func (r manager) Del(workerId int, conn *connection) {
	if c, ok := r[workerId]; ok {
		c.Del(conn)
	}
}

// Manager 管理所有的工作进程
var Manager manager

func init() {
	Manager = manager{}
	for i := MinWorkerId; i <= MaxWorkerId; i++ {
		Manager[i] = &collect{conn: []*connection{}, index: 0, lock: sync.Mutex{}}
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
