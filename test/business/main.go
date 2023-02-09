package main

import (
	"fmt"
	"github.com/lesismal/nbio/logging"
	"html/template"
	"net"
	"net/http"
	"netsvr/configs"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/pkg/quit"
	"netsvr/test/business/cmd/client"
	"netsvr/test/business/cmd/svr"
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
		for cmd, name := range protocol.CmdName {
			data[name] = int(cmd)
		}
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
	//注册网关发送过来的命令的处理函数
	processor.RegisterSvrCmd(internalProtocol.Cmd_ConnOpen, svr.ConnOpen)
	processor.RegisterSvrCmd(internalProtocol.Cmd_ConnClose, svr.ConnClose)
	processor.RegisterSvrCmd(internalProtocol.Cmd_NetSvrStatusRe, svr.NetSvrStatus)
	processor.RegisterSvrCmd(internalProtocol.Cmd_InfoRe, svr.InfoRe)
	processor.RegisterSvrCmd(internalProtocol.Cmd_TopicsUniqIdCountRe, svr.TopicsUniqIdCount)
	processor.RegisterSvrCmd(internalProtocol.Cmd_TopicUniqIdsRe, svr.TopicUniqIds)
	processor.RegisterSvrCmd(internalProtocol.Cmd_TotalUniqIdsRe, svr.TotalUniqIds)
	//注册用户发送过来的命令的处理函数
	processor.RegisterClientCmd(protocol.RouterBroadcast, client.Broadcast)
	processor.RegisterClientCmd(protocol.RouterLogin, client.Login)
	processor.RegisterClientCmd(protocol.RouterLogout, client.Logout)
	processor.RegisterClientCmd(protocol.RouterMulticastForUserId, client.MulticastForUserId)
	processor.RegisterClientCmd(protocol.RouterMulticastForUniqId, client.MulticastForUniqId)
	processor.RegisterClientCmd(protocol.RouterNetSvrStatus, client.NetSvrStatus)
	processor.RegisterClientCmd(protocol.RouterPublish, client.Publish)
	processor.RegisterClientCmd(protocol.RouterSingleCastForUserId, client.SingleCastForUserId)
	processor.RegisterClientCmd(protocol.RouterSingleCastForUniqId, client.SingleCastForUniqId)
	processor.RegisterClientCmd(protocol.RouterSubscribe, client.Subscribe)
	processor.RegisterClientCmd(protocol.RouterTopicsUniqIdCount, client.TopicsUniqIdCount)
	processor.RegisterClientCmd(protocol.RouterTopicUniqIds, client.TopicUniqIds)
	processor.RegisterClientCmd(protocol.RouterTotalUniqIds, client.TotalUniqIds)
	processor.RegisterClientCmd(protocol.RouterTopicList, client.TopicList)
	processor.RegisterClientCmd(protocol.RouterUnsubscribe, client.Unsubscribe)
	processor.RegisterClientCmd(protocol.RouterForceOfflineForUserId, client.ForceOfflineForUserId)
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
