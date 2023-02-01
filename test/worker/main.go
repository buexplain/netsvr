package main

import (
	"github.com/lesismal/nbio/logging"
	"net"
	toWorkerRouter "netsvr/internal/protocol/toWorker/router"
	"netsvr/pkg/quit"
	"netsvr/test/worker/cmd/client"
	"netsvr/test/worker/cmd/svr"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"os"
	"time"
)

func init() {
	logging.SetLevel(logging.LevelDebug)
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8888")
	if err != nil {
		logging.Error("连接服务端失败，%v", err)
		os.Exit(1)
	}
	processor := connProcessor.NewConnProcessor(conn, 1)
	//注册工作进程
	if err := processor.RegisterWorker(); err != nil {
		logging.Error("注册工作进程失败 %v", err)
		os.Exit(1)
	}
	logging.Debug("注册工作进程 %d ok", processor.GetWorkerId())
	//注册网关发送过来的命令
	processor.RegisterSvrCmd(toWorkerRouter.Cmd_ConnOpen, svr.ConnOpen)
	processor.RegisterSvrCmd(toWorkerRouter.Cmd_ConnClose, svr.ConnClose)
	processor.RegisterSvrCmd(toWorkerRouter.Cmd_RespNetSvrStatus, svr.RespNetSvrStatus)
	processor.RegisterSvrCmd(toWorkerRouter.Cmd_RespSessionInfo, svr.RespSessionInfo)
	processor.RegisterSvrCmd(toWorkerRouter.Cmd_RespTopicsConnCount, svr.RespTopicsConnCount)
	processor.RegisterSvrCmd(toWorkerRouter.Cmd_RespTopicsSessionId, svr.RespTopicsSessionId)
	processor.RegisterSvrCmd(toWorkerRouter.Cmd_RespTotalSessionId, svr.RespTotalSessionId)
	//注册用户发送过来的命令
	processor.RegisterClientCmd(protocol.RouterBroadcast, client.Broadcast)
	processor.RegisterClientCmd(protocol.RouterLogin, client.Login)
	processor.RegisterClientCmd(protocol.RouterLogout, client.Logout)
	processor.RegisterClientCmd(protocol.RouterMulticast, client.Multicast)
	processor.RegisterClientCmd(protocol.RouterNetSvrStatus, client.NetSvrStatus)
	processor.RegisterClientCmd(protocol.RouterPublish, client.Publish)
	processor.RegisterClientCmd(protocol.RouterSingleCast, client.SingleCast)
	processor.RegisterClientCmd(protocol.RouterSubscribe, client.Subscribe)
	processor.RegisterClientCmd(protocol.RouterTopicsConnCount, client.TopicsConnCount)
	processor.RegisterClientCmd(protocol.RouterTopicsSessionId, client.TopicsSessionId)
	processor.RegisterClientCmd(protocol.RouterTotalSessionId, client.TotalSessionId)
	processor.RegisterClientCmd(protocol.RouterUnsubscribe, client.Unsubscribe)
	processor.RegisterClientCmd(protocol.RouterUpdateSessionUserInfo, client.UpdateSessionUserInfo)
	//心跳
	quit.Wg.Add(1)
	go processor.LoopHeartbeat()
	//循环处理网关的请求
	for i := 0; i < 10; i++ {
		quit.Wg.Add(1)
		go processor.LoopCmd(i)
	}
	//循环写
	quit.Wg.Add(1)
	go processor.LoopSend()
	//循环读
	go processor.LoopReceive()
	//开始关闭进程
	select {
	case <-quit.ClosedCh:
		//及时打印关闭进程的日志，避免使用者认为进程无反应，直接强杀进程
		logging.Info("开始工作进程: pid --> %d 原因 --> %s", os.Getpid(), quit.GetReason())
		//取消注册工作进程
		processor.UnregisterWorker()
		//发出关闭信号，通知所有协程
		quit.Cancel()
		//等待协程退出
		quit.Wg.Wait()
		//关闭连接，这里暂停一下，等待数据发送出去
		time.Sleep(time.Millisecond * 100)
		processor.Close()
		logging.Info("关闭进程成功: pid --> %d", os.Getpid())
		os.Exit(0)
	}
}
