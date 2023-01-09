package worker

import (
	"github.com/buexplain/netsvr/internal/worker/manager"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/lesismal/nbio/logging"
	"net"
	"time"
)

type Server struct {
	listener net.Listener
}

func (r *Server) Start() {
	defer func() {
		logging.Debug("Worker tcp stop accept")
	}()
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
		select {
		case <-quit.Ctx.Done():
			//进程即将停止，不再受理新的连接
			_ = conn.Close()
			continue
		default:
			c := manager.NewConnection(conn)
			go c.Read()
			quit.Wg.Add(1)
			go c.Send()
		}
	}
}

var server *Server

func Start() {
	listen, err := net.Listen("tcp", "localhost:8888")
	if err != nil {
		return
	}
	server = &Server{
		listener: listen,
	}
	server.Start()
}

func Shutdown() {
	err := server.listener.Close()
	if err != nil {
		logging.Error("Worker tcp shutdown failed: %v", err)
		return
	}
	logging.Info("Worker tcp shutdown")
}
