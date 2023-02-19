package customer

import (
	"bytes"
	"context"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"net/http"
	"netsvr/configs"
	"netsvr/internal/customer/info"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/heartbeat"
	"netsvr/internal/metrics"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/quit"
	"netsvr/pkg/utils"
	"strings"
)

var server *nbhttp.Server

func Start() {
	mux := &http.ServeMux{}
	mux.HandleFunc(configs.Config.CustomerHandlePattern, onWebsocket)
	config := nbhttp.Config{
		Network: "tcp",
		Addrs:   []string{configs.Config.CustomerListenAddress},
		Handler: mux,
		MaxLoad: configs.Config.CustomerMaxOnlineNum,
		//TODO 测试更多的配置信息
	}
	config.Name = "customer"
	server = nbhttp.NewServer(config)
	err := server.Start()
	if err != nil {
		logging.Error("Customer websocket start failed: %v", err)
		return
	}
	logging.Info("Customer websocket start")
}

func Shutdown() {
	err := server.Shutdown(context.Background())
	if err != nil {
		logging.Error("Customer websocket shutdown failed: %v", err)
		return
	}
	logging.Info("Customer websocket shutdown")
}

func onWebsocket(w http.ResponseWriter, r *http.Request) {
	select {
	case <-quit.Ctx.Done():
		//进程即将关闭，不再受理新的连接
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(http.StatusText(http.StatusServiceUnavailable)))
		return
	default:
	}
	upgrade := websocket.NewUpgrader()
	upgrade.KeepaliveTime = configs.Config.CustomerReadDeadline
	upgrade.CheckOrigin = func(r *http.Request) bool {
		if len(configs.Config.CustomerAllowOrigin) == 0 {
			return true
		}
		origin := r.Header.Get("Origin")
		for _, v := range configs.Config.CustomerAllowOrigin {
			if strings.Contains(origin, v) {
				return true
			}
		}
		return false
	}
	upgrade.OnOpen(func(conn *websocket.Conn) {
		//统计打开连接次数
		metrics.Registry[metrics.ItemCustomerConnOpen].Meter.Mark(1)
		//分配uniqId，并将添加到管理器中
		uniqId := utils.UniqId()
		session := info.NewInfo(uniqId)
		conn.SetSession(session)
		manager.Manager.Set(uniqId, conn)
		logging.Debug("Customer websocket open, info: %#v", session)
		//获取处理连接打开的worker
		workerId := workerManager.GetProcessConnOpenWorkerId()
		worker := workerManager.Manager.Get(workerId)
		if worker == nil {
			logging.Debug("Not found process conn open business by id: %d", workerId)
			return
		}
		//连接打开消息回传给business
		co := &protocol.ConnOpen{}
		co.UniqId = session.GetUniqId()
		router := &protocol.Router{}
		router.Cmd = protocol.Cmd_ConnOpen
		router.Data, _ = proto.Marshal(co)
		data, _ := proto.Marshal(router)
		worker.Send(data)
		//统计转发到business的次数与字节数
		metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(len(data) + 4)) //加上4字节，是因为tcp包头的缘故
	})
	upgrade.OnClose(func(conn *websocket.Conn, err error) {
		//统计关闭连接次数
		metrics.Registry[metrics.ItemCustomerConnClose].Meter.Mark(1)
		session, ok := conn.Session().(*info.Info)
		if !ok {
			return
		}
		topics, uniqId, userSession := session.Clear()
		if uniqId == "" {
			//当前连接已经被清空了uniqId，无需进行接下来的逻辑
			return
		}
		logging.Debug("Customer websocket close, info: %#v", session)
		//从连接管理器中删除
		manager.Manager.Del(uniqId)
		//删除订阅关系
		topic.Topic.Del(topics, uniqId)
		//连接关闭消息回传给business
		workerId := workerManager.GetProcessConnCloseWorkerId()
		worker := workerManager.Manager.Get(workerId)
		if worker == nil {
			logging.Debug("Not found process conn close business by id: %d", workerId)
			return
		}
		//转发数据到business
		cl := &protocol.ConnClose{}
		cl.UniqId = uniqId
		cl.Session = userSession
		router := &protocol.Router{}
		router.Cmd = protocol.Cmd_ConnClose
		router.Data, _ = proto.Marshal(cl)
		data, _ := proto.Marshal(router)
		worker.Send(data)
		//统计转发到business的次数与字节数
		metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(len(data) + 4)) //加上4字节，是因为tcp包头的缘故
	})
	upgrade.OnMessage(func(conn *websocket.Conn, messageType websocket.MessageType, data []byte) {
		//检查是否为心跳消息
		if bytes.Equal(data, heartbeat.PingMessage) {
			//响应客户端心跳
			err := conn.WriteMessage(websocket.TextMessage, heartbeat.PongMessage)
			if err != nil {
				_ = conn.Close()
			}
			metrics.Registry[metrics.ItemCustomerHeartbeat].Meter.Mark(1)
			return
		} else if messageType == websocket.PingMessage {
			//响应客户端心跳
			err := conn.WriteMessage(websocket.PongMessage, heartbeat.PongMessage)
			if err != nil {
				_ = conn.Close()
			}
			metrics.Registry[metrics.ItemCustomerHeartbeat].Meter.Mark(1)
			return
		}
		session, ok := conn.Session().(*info.Info)
		if !ok {
			return
		}
		//读取前三个字节，转成business的服务编号
		workerId := utils.BytesToInt(data, 3)
		worker := workerManager.Manager.Get(workerId)
		if worker == nil {
			logging.Debug("Not found business by id: %d", workerId)
			return
		}
		//编码数据成business需要的格式
		tf := &protocol.Transfer{}
		tf.Data = data[3:]
		session.GetToProtocolTransfer(tf)
		router := &protocol.Router{}
		router.Cmd = protocol.Cmd_Transfer
		router.Data, _ = proto.Marshal(tf)
		//转发数据到business
		data, _ = proto.Marshal(router)
		worker.Send(data)
		//统计转发到business的次数与字节数
		metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(len(data) + 4)) //加上4字节，是因为tcp包头的缘故
	})
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		logging.Error("Customer websocket upgrade failed: %v", err)
		return
	}
	logging.Debug("Customer websocket upgrade ok, remoteAddr: %s", conn.RemoteAddr())
}
