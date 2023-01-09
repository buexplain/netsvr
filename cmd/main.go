package main

import (
	"github.com/buexplain/netsvr/internal/customer"
	"github.com/buexplain/netsvr/internal/worker"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/lesismal/nbio/logging"
	"os"
)

func main() {
	go func() {
		//及时打印关闭进程的日志，避免使用者认为进程无反应，直接强杀进程
		<-quit.ClosedCh
		logging.Info("开始关闭进程: pid --> %d 原因 --> %s", os.Getpid(), quit.GetReason())
	}()
	logging.SetLevel(logging.LevelDebug)
	go worker.Start()
	go customer.Start()
	select {
	case <-quit.ClosedCh:
		//先发出关闭信号，通知所有协程
		quit.Cancel()
		//等待协程退出
		quit.Wg.Wait()
		//关闭工作进程服务器
		worker.Shutdown()
		//关闭客户服务器
		customer.Shutdown()
		logging.Info("关闭进程成功: pid --> %d", os.Getpid())
		os.Exit(0)
	}
}
