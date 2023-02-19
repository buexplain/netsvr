package main

import (
	"fmt"
	"github.com/lesismal/nbio/logging"
	"html/template"
	"net"
	"net/http"
	"netsvr/configs"
	"netsvr/internal/heartbeat"
	"netsvr/pkg/quit"
	"netsvr/test/business/cmd"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"os"
)

func init() {
	logging.SetLevel(logging.LevelDebug)
}

// 输出html客户端
func clientServer() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		t, err := template.New("client.html").Delims("{!", "!}").ParseFiles(configs.RootPath + "test/business/client.html")
		if err != nil {
			logging.Error("模板解析失败：%s", err)
			return
		}
		data := map[string]interface{}{}
		//注入连接地址
		data["conn"] = fmt.Sprintf("ws://%s%s", configs.Config.CustomerListenAddress, configs.Config.CustomerHandlePattern)
		//把所有的命令注入到客户端
		for c, name := range protocol.CmdName {
			data[name] = int(c)
		}
		data["pingMessage"] = string(heartbeat.PingMessage)
		data["pongMessage"] = string(heartbeat.PongMessage)
		err = t.Execute(writer, data)
		if err != nil {
			logging.Error("模板输出失败：%s", err)
			return
		}
	})
	logging.Info("点击访问客户端：http://127.0.0.1:6062/")
	_ = http.ListenAndServe("127.0.0.1:6062", nil)
}

func main() {
	conn, err := net.Dial("tcp", configs.Config.WorkerListenAddress)
	if err != nil {
		logging.Error("连接服务端失败，%v", err)
		os.Exit(1)
	}
	//启动html客户端的服务器
	go clientServer()
	processor := connProcessor.NewConnProcessor(conn, 1)
	//注册到worker
	if err := processor.RegisterWorker(); err != nil {
		logging.Error("注册到worker失败 %v", err)
		os.Exit(1)
	}
	logging.Debug("注册到worker %d ok", processor.GetWorkerId())
	//注册各种回调函数
	cmd.CheckOnline.Init(processor)
	cmd.Broadcast.Init(processor)
	cmd.Multicast.Init(processor)
	cmd.SingleCast.Init(processor)
	cmd.ConnSwitch.Init(processor)
	cmd.Sign.Init(processor)
	cmd.ForceOffline.Init(processor)
	cmd.Topic.Init(processor)
	cmd.UniqId.Init(processor)
	cmd.Metrics.Init(processor)
	//心跳
	quit.Wg.Add(1)
	go processor.LoopHeartbeat()
	//循环处理worker发来的指令
	for i := 0; i < 2; i++ {
		quit.Wg.Add(1)
		go processor.LoopCmd(i)
	}
	//循环写
	quit.Wg.Add(1)
	go processor.LoopSend()
	//循环读
	quit.Wg.Add(1)
	go processor.LoopReceive()
	//开始关闭进程
	select {
	case <-quit.ClosedCh:
		//及时打印关闭进程的日志，避免使用者认为进程无反应，直接强杀进程
		logging.Info("开始关闭business进程: pid --> %d 原因 --> %s", os.Getpid(), quit.GetReason())
		//取消注册
		processor.UnregisterWorker()
		//发出关闭信号，通知所有协程
		quit.Cancel()
		//等待协程退出
		quit.Wg.Wait()
		processor.Close()
		logging.Info("关闭business进程成功: pid --> %d", os.Getpid())
		os.Exit(0)
	}
}
