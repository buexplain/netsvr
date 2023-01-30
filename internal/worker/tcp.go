package worker

import (
	"github.com/lesismal/nbio/logging"
	"net"
	"netsvr/configs"
	"netsvr/internal/cmd"
	"netsvr/internal/protocol/toServer/router"
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
			c.RegisterCmd(router.Cmd_RegisterWorker, cmd.RegisterWorker)
			c.RegisterCmd(router.Cmd_UnregisterWorker, cmd.UnregisterWorker)
			c.RegisterCmd(router.Cmd_Broadcast, cmd.Broadcast)
			c.RegisterCmd(router.Cmd_Multicast, cmd.Multicast)
			c.RegisterCmd(router.Cmd_MulticastByBitmap, cmd.MulticastByBitmap)
			c.RegisterCmd(router.Cmd_Publish, cmd.Publish)
			c.RegisterCmd(router.Cmd_SetSessionUser, cmd.SetSessionUser)
			c.RegisterCmd(router.Cmd_SetUserLoginStatus, cmd.SetUserLoginStatus)
			c.RegisterCmd(router.Cmd_SingleCast, cmd.SingleCast)
			c.RegisterCmd(router.Cmd_Subscribe, cmd.Subscribe)
			c.RegisterCmd(router.Cmd_Unsubscribe, cmd.Unsubscribe)
			c.RegisterCmd(router.Cmd_ReqTotalSessionId, cmd.ReqTotalSessionId)
			c.RegisterCmd(router.Cmd_ReqTopicsSessionId, cmd.ReqTopicsSessionId)
			c.RegisterCmd(router.Cmd_ReqTopicsConnCount, cmd.ReqTopicsConnCount)
			c.RegisterCmd(router.Cmd_ReqSessionInfo, cmd.ReqSessionInfo)
			c.RegisterCmd(router.Cmd_ReqNetSvrStatus, cmd.ReqNetSvrStatus)
			for i := 0; i < configs.Config.WorkerConsumer; i++ {
				quit.Wg.Add(1)
				go c.LoopCmd(i)
			}
			quit.Wg.Add(2)
			go c.LoopRead()
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
