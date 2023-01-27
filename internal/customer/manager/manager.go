package manager

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"sync"
)

type collect struct {
	conn map[uint32]*websocket.Conn
	lock sync.RWMutex
}

func (r *collect) Get(sessionId uint32) *websocket.Conn {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if c, ok := r.conn[sessionId]; ok {
		return c
	}
	return nil
}

func (r *collect) Set(sessionId uint32, conn *websocket.Conn) {
	r.lock.Lock()
	r.conn[sessionId] = conn
	r.lock.Unlock()
}

func (r *collect) Del(sessionId uint32) {
	r.lock.Lock()
	delete(r.conn, sessionId)
	r.lock.Unlock()
}

const managerLen = 8

type manager [managerLen]*collect

func (r manager) Get(sessionId uint32) *websocket.Conn {
	//根据session id取模，将连接分布在不同的集合中
	//1. 避免单个map存储的连接数太多，导致gc抖动
	//2. 避免大锁
	index := sessionId % managerLen
	return r[index].Get(sessionId)
}

func (r manager) Set(sessionId uint32, conn *websocket.Conn) {
	index := sessionId % managerLen
	r[index].Set(sessionId, conn)
}

func (r manager) Del(sessionId uint32) {
	index := sessionId % managerLen
	r[index].Del(sessionId)
}

var Manager manager

func init() {
	Manager = manager{}
	for i := 0; i < len(Manager); i++ {
		Manager[i] = &collect{conn: map[uint32]*websocket.Conn{}, lock: sync.RWMutex{}}
	}
}
