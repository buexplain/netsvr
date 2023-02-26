package main

import (
	"github.com/lesismal/nbio/logging"
	"net/http"
	_ "net/http/pprof"
	"netsvr/configs"
	"netsvr/internal/customer"
	"netsvr/internal/worker"
	"netsvr/pkg/quit"
	"os"
)

func main() {
	logging.SetLevel(configs.Config.GetLogLevel())
	go func() {
		defer func() {
			_ = recover()
		}()
		logging.Info("Pprof http start http://127.0.0.1:6062/debug/pprof")
		_ = http.ListenAndServe("127.0.0.1:6062", nil)
	}()
	go worker.Start()
	go customer.Start()
	select {
	case <-quit.ClosedCh:
		//及时打印关闭进程的日志，避免使用者认为进程无反应，直接强杀进程
		logging.Info("开始关闭网关进程: pid --> %d 原因 --> %s", os.Getpid(), quit.GetReason())
		//通知所有协程开始退出
		quit.Cancel()
		//等待协程退出
		quit.Wg.Wait()
		//关闭worker服务器
		worker.Shutdown()
		//关闭customer服务器
		customer.Shutdown()
		logging.Info("关闭网关进程成功: pid --> %d", os.Getpid())
		os.Exit(0)
	}
}
