package manager

import (
	"net"
	"sync"
)

type collect struct {
	conn  []net.Conn
	index int
	lock  sync.RWMutex
}

func (r *collect) Get() net.Conn {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if len(r.conn) == 0 {
		return nil
	}
	r.index++
	if r.index >= len(r.conn) {
		r.index = 0
	}
	return r.conn[r.index]
}

func (r *collect) Set(conn net.Conn) {
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

func (r *collect) Del(conn net.Conn) {
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

func (r manager) Get(workerId int) net.Conn {
	if c, ok := r[workerId]; ok {
		return c.Get()
	}
	return nil
}

func (r manager) Set(workerId int, conn net.Conn) {
	if c, ok := r[workerId]; ok {
		c.Set(conn)
	}
}

func (r manager) Del(workerId int, conn net.Conn) {
	if c, ok := r[workerId]; ok {
		c.Del(conn)
	}
}

var Manager manager

func init() {
	Manager = manager{}
	for i := MinWorkerId; i <= MaxWorkerId; i++ {
		Manager[i] = &collect{conn: []net.Conn{}, index: 0, lock: sync.RWMutex{}}
	}
}
