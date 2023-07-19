package connPool

import (
	"net"
	"time"
)

type ConnPool struct {
	pool      chan net.Conn
	heartbeat func(conn net.Conn) bool
	factory   func() net.Conn
}

func NewConnPool(size int, factory func() net.Conn, heartbeat func(conn net.Conn) bool, heartbeatDuration time.Duration) *ConnPool {
	cp := &ConnPool{
		pool:      make(chan net.Conn, size),
		heartbeat: heartbeat,
		factory:   factory,
	}
	for i := 0; i < size; i++ {
		cp.pool <- factory()
	}
	cp.loopHeartbeat(heartbeatDuration)
	return cp
}

func (r *ConnPool) loopHeartbeat(heartbeatDuration time.Duration) {
	go func() {
		t := time.NewTicker(heartbeatDuration)
		defer t.Stop()
		for {
			<-t.C
			for i := 0; i < cap(r.pool); i++ {
				conn := <-r.pool
				if !r.heartbeat(conn) {
					conn = r.factory()
				}
				r.pool <- conn
			}
		}
	}()
}

func (r *ConnPool) Factory() net.Conn {
	return r.factory()
}

func (r *ConnPool) Get() net.Conn {
	return <-r.pool
}

func (r *ConnPool) Put(conn net.Conn) {
	r.pool <- conn
}
