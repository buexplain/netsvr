package worker

import (
	"github.com/lesismal/nbio/logging"
	"net"
	"netsvr/configs"
	"netsvr/internal/cmd"
	"netsvr/internal/protocol"
	"netsvr/internal/worker/manager"
	"netsvr/pkg/quit"
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
			c := manager.NewConnProcessor(conn)
			c.RegisterCmd(protocol.Cmd_Register, cmd.Register)
			c.RegisterCmd(protocol.Cmd_Unregister, cmd.Unregister)
			c.RegisterCmd(protocol.Cmd_Broadcast, cmd.Broadcast)
			c.RegisterCmd(protocol.Cmd_Multicast, cmd.Multicast)
			c.RegisterCmd(protocol.Cmd_MulticastByBitmap, cmd.MulticastByBitmap)
			c.RegisterCmd(protocol.Cmd_Publish, cmd.Publish)
			c.RegisterCmd(protocol.Cmd_UpSessionUserInfo, cmd.UpSessionUserInfo)
			c.RegisterCmd(protocol.Cmd_SetUserLoginStatus, cmd.SetUserLoginStatus)
			c.RegisterCmd(protocol.Cmd_SingleCast, cmd.SingleCast)
			c.RegisterCmd(protocol.Cmd_Subscribe, cmd.Subscribe)
			c.RegisterCmd(protocol.Cmd_Unsubscribe, cmd.Unsubscribe)
			c.RegisterCmd(protocol.Cmd_TotalSessionId, cmd.TotalSessionId)
			c.RegisterCmd(protocol.Cmd_TopicsSessionId, cmd.TopicsSessionId)
			c.RegisterCmd(protocol.Cmd_TopicsConnCount, cmd.TopicsConnCount)
			c.RegisterCmd(protocol.Cmd_SessionInfo, cmd.SessionInfo)
			c.RegisterCmd(protocol.Cmd_NetSvrStatus, cmd.NetSvrStatus)
			for i := 0; i < configs.Config.WorkerConsumer; i++ {
				quit.Wg.Add(1)
				go c.LoopCmd(i)
			}
			quit.Wg.Add(2)
			go c.LoopReceive()
			go c.LoopSend()
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
