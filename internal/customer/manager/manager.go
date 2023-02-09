package manager

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"hash/adler32"
	"sync"
	"unsafe"
)

type collect struct {
	conn map[string]*websocket.Conn
	mux  sync.RWMutex
}

func (r *collect) Len() int {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return len(r.conn)
}

func (r *collect) GetUniqIds(uniqIds *[]string) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	for uniqId := range r.conn {
		*uniqIds = append(*uniqIds, uniqId)
	}
}

func (r *collect) Has(uniqId string) bool {
	r.mux.RLock()
	defer r.mux.RUnlock()
	_, ok := r.conn[uniqId]
	return ok
}

func (r *collect) Get(uniqId string) *websocket.Conn {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if c, ok := r.conn[uniqId]; ok {
		return c
	}
	return nil
}

func (r *collect) Set(uniqId string, conn *websocket.Conn) {
	r.mux.Lock()
	r.conn[uniqId] = conn
	r.mux.Unlock()
}

func (r *collect) Del(uniqId string) {
	r.mux.Lock()
	delete(r.conn, uniqId)
	r.mux.Unlock()
}

const managerLen = 8

type manager [managerLen]*collect

func (r manager) index(uniqId string) uint32 {
	return adler32.Checksum(unsafe.Slice(unsafe.StringData(uniqId), len(uniqId))) % managerLen
}

func (r manager) Has(uniqId string) bool {
	return r[r.index(uniqId)].Has(uniqId)
}

func (r manager) Get(uniqId string) *websocket.Conn {
	return r[r.index(uniqId)].Get(uniqId)
}

func (r manager) Set(uniqId string, conn *websocket.Conn) {
	r[r.index(uniqId)].Set(uniqId, conn)
}

func (r manager) Del(uniqId string) {
	r[r.index(uniqId)].Del(uniqId)
}

var Manager manager

func init() {
	for i := 0; i < len(Manager); i++ {
		Manager[i] = &collect{conn: make(map[string]*websocket.Conn, 1000), mux: sync.RWMutex{}}
	}
}
