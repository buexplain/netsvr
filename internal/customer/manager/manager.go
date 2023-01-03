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
	if c, ok := r.conn[sessionId]; ok {
		r.lock.RUnlock()
		return c
	}
	r.lock.RUnlock()
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
