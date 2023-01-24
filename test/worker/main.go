package main

import (
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/buexplain/netsvr/test/worker/connProcessor"
	"github.com/lesismal/nbio/logging"
	"net"
	"os"
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
	go func() {
		<-quit.ClosedCh
		logging.Info("开始工作进程: pid --> %d 原因 --> %s", os.Getpid(), quit.GetReason())
		//取消注册工作进程
		processor.UnregisterWorker()
		//关闭连接
		processor.Close()
	}()
	go processor.LoopSend()
	go processor.Heartbeat()
	processor.LoopRead()
}
