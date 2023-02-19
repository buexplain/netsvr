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
			c.RegisterCmd(protocol.Cmd_InfoUpdate, cmd.InfoUpdate)
			c.RegisterCmd(protocol.Cmd_InfoDelete, cmd.InfoDelete)
			c.RegisterCmd(protocol.Cmd_ForceOffline, cmd.ForceOffline)
			c.RegisterCmd(protocol.Cmd_Multicast, cmd.Multicast)
			c.RegisterCmd(protocol.Cmd_TopicPublish, cmd.TopicPublish)
			c.RegisterCmd(protocol.Cmd_SingleCast, cmd.SingleCast)
			c.RegisterCmd(protocol.Cmd_TopicSubscribe, cmd.TopicSubscribe)
			c.RegisterCmd(protocol.Cmd_TopicUnsubscribe, cmd.TopicUnsubscribe)
			c.RegisterCmd(protocol.Cmd_TopicDelete, cmd.TopicDelete)
			c.RegisterCmd(protocol.Cmd_UniqIdList, cmd.UniqIdList)
			c.RegisterCmd(protocol.Cmd_UniqIdCount, cmd.UniqIdCount)
			c.RegisterCmd(protocol.Cmd_TopicUniqIdList, cmd.TopicUniqIdList)
			c.RegisterCmd(protocol.Cmd_TopicUniqIdCount, cmd.TopicUniqIdCount)
			c.RegisterCmd(protocol.Cmd_TopicCount, cmd.TopicCount)
			c.RegisterCmd(protocol.Cmd_TopicList, cmd.TopicList)
			c.RegisterCmd(protocol.Cmd_Info, cmd.Info)
			c.RegisterCmd(protocol.Cmd_Metrics, cmd.Metrics)
			c.RegisterCmd(protocol.Cmd_CheckOnline, cmd.CheckOnline)
			//启动三条协程，负责处理命令、读取数据、写入数据、更多的处理命令协程，business在注册的时候可以自定义，要求worker进行开启
			quit.Wg.Add(3)
			go c.LoopCmd(0)
			go c.LoopReceive()
			go c.LoopSend()
		}
	}
}

var server *Server

func Start() {
	listen, err := net.Listen("tcp", configs.Config.WorkerListenAddress)
	if err != nil {
		return
	}
	server = &Server{
		listener: listen,
	}
	logging.Info("Worker tcp start")
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
