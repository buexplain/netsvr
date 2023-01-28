package main

import (
	"github.com/lesismal/nbio/logging"
	"net"
	"netsvr/pkg/quit"
	"netsvr/test/worker/connProcessor"
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
	quit.Wg.Add(2)
	go processor.LoopSend()
	go processor.LoopHeartbeat()
	go processor.LoopRead()
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
