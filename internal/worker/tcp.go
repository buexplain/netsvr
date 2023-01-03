package worker

import (
	"net"
	"sync"
	"time"
)

type Server struct {
	listener net.Listener
	sessions sync.Map
	stopOnce sync.Once
}

func (r *Server) Start() {
	var delay int64 = 0
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if delay == 0 {
					delay = 15
				} else {
					delay *= 2
				}
				if delay > 1000 {
					delay = 1000
				}
				time.Sleep(time.Millisecond * time.Duration(delay))
				continue
			}
			return
		}
		go r.handleNewConn(conn)
	}
}

func (r *Server) handleNewConn(conn net.Conn) {
	for {

	}
}
