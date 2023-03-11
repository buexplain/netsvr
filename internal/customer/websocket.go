// Package customer 客户连接维持模块
// 负责承载客户的websocket连接、维护客户的订阅、保持客户连接的session数据
package customer

import (
	"bytes"
	"context"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"net/http"
	"netsvr/configs"
	"netsvr/internal/customer/info"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/limit"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/utils"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/heartbeat"
	protocol2 "netsvr/pkg/protocol"
	"netsvr/pkg/quit"
	"strings"
)

var server *nbhttp.Server
var serviceBusy = []byte("Service Busy")
var serviceRestarting = []byte("Service Restarting")
var dataTooLarge = []byte("Data too large")

func Start() {
	mux := &http.ServeMux{}
	mux.HandleFunc(configs.Config.Customer.HandlePattern, onWebsocket)
	config := nbhttp.Config{
		Network: "tcp",
		Addrs:   []string{configs.Config.Customer.ListenAddress},
		Handler: mux,
		MaxLoad: configs.Config.Customer.MaxOnlineNum,
	}
	config.Name = "customer"
	server = nbhttp.NewServer(config)
	err := server.Start()
	if err != nil {
		log.Logger.Error().Err(err).Msg("Customer websocket start failed")
		return
	}
	log.Logger.Info().Msg("Customer websocket start")
}

func Shutdown() {
	err := server.Shutdown(context.Background())
	if err != nil {
		log.Logger.Error().Err(err).Msg("Customer websocket grace shutdown failed")
		return
	}
	log.Logger.Info().Msg("Customer websocket grace shutdown")
}

func onWebsocket(w http.ResponseWriter, r *http.Request) {
	var workerId int
	select {
	case <-quit.Ctx.Done():
		//进程即将关闭，不再受理新的连接
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(serviceRestarting)
		return
	default:
		//限流检查
		workerId = workerManager.GetProcessConnOpenWorkerId()
		//之所以要判断workerId大于0，是因为有可能业务方并不关心连接的打开信息，连接打开信息不会传递到业务方，则不必限流
		if workerId > 0 && limit.Manager.Allow(workerId) == false {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write(serviceBusy)
			return
		}
	}
	upgrade := websocket.NewUpgrader()
	upgrade.KeepaliveTime = configs.Config.Customer.ReadDeadline
	upgrade.CheckOrigin = checkOrigin
	upgrade.SetPingHandler(pingMessageHandler)
	upgrade.SetPongHandler(pongMessageHandler)
	upgrade.OnOpen(nil)
	upgrade.OnClose(onClose)
	upgrade.OnMessage(onMessage)
	//处理websocket子协议
	upgrade.Subprotocols = nil
	subProtocols := utils.ParseSubProtocols(r)
	var responseHeader http.Header
	if subProtocols != nil {
		//返回任意一个子协议，保证连接升级成功
		responseHeader = http.Header{}
		responseHeader["Sec-Websocket-Protocol"] = subProtocols[0:1]
	}
	//开始升级
	conn, err := upgrade.Upgrade(w, r, responseHeader)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Customer websocket upgrade failed")
		return
	}
	//升级成功
	wsConn := conn.(*websocket.Conn)
	//统计打开连接次数
	metrics.Registry[metrics.ItemCustomerConnOpen].Meter.Mark(1)
	//分配uniqId，并将添加到管理器中
	uniqId := utils.UniqId()
	session := info.NewInfo(uniqId)
	wsConn.SetSession(session)
	manager.Manager.Set(uniqId, wsConn)
	log.Logger.Debug().Str("uniqId", uniqId).Msg("Customer websocket open")
	//获取能够处理连接打开信息的business
	worker := workerManager.Manager.Get(workerId)
	if worker == nil {
		log.Logger.Debug().Int("workerId", workerId).Msg("Not found process conn open business")
		return
	}
	//连接打开消息回传给business
	co := &protocol2.ConnOpen{}
	co.SubProtocol = subProtocols
	co.XForwardedFor = r.Header.Get("X-Forwarded-For")
	co.XRealIP = r.Header.Get(configs.Config.Customer.XRealIP)
	co.RemoteIP = strings.Split(conn.RemoteAddr().String(), ":")[0]
	co.RawQuery = r.URL.RawQuery
	co.UniqId = uniqId
	router := &protocol2.Router{}
	router.Cmd = protocol2.Cmd_ConnOpen
	router.Data, _ = proto.Marshal(co)
	data, _ := proto.Marshal(router)
	worker.Send(data)
	//统计转发到business的次数与字节数
	metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(len(data) + 4)) //加上4字节，是因为tcp包头的缘故
}

// ping客户端心跳帧
func pongMessageHandler(_ *websocket.Conn, _ string) {
}

// pong客户端心跳帧
func pingMessageHandler(conn *websocket.Conn, _ string) {
	//响应客户端心跳
	err := conn.WriteMessage(websocket.PongMessage, heartbeat.PongMessage)
	if err != nil {
		_ = conn.Close()
	}
	metrics.Registry[metrics.ItemCustomerHeartbeat].Meter.Mark(1)
}

func onClose(conn *websocket.Conn, _ error) {
	//分配了session才算是真正打开过的连接
	session, ok := conn.Session().(*info.Info)
	if !ok {
		return
	}
	//统计关闭连接次数
	metrics.Registry[metrics.ItemCustomerConnClose].Meter.Mark(1)
	session.MuxLock()
	//关闭info
	session.Close()
	//断开info与conn的关系，便于快速gc
	conn.SetSession(nil)
	//清空info，并返回相关数据
	topics, uniqId, userSession := session.Clear()
	if uniqId == "" {
		session.MuxUnLock()
		//当前连接已经被清空了uniqId，无需进行接下来的逻辑
		return
	}
	//从连接管理器中删除
	manager.Manager.Del(uniqId)
	//删除订阅关系
	topic.Topic.DelByMap(topics, uniqId, "")
	//释放锁
	session.MuxUnLock()
	log.Logger.Debug().Interface("topics", topics).Str("uniqId", uniqId).Str("session", userSession).Msg("Customer websocket close")
	//连接关闭消息回传给business
	workerId := workerManager.GetProcessConnCloseWorkerId()
	worker := workerManager.Manager.Get(workerId)
	if worker == nil {
		log.Logger.Debug().Int("workerId", workerId).Msg("Not found process conn close business")
		return
	}
	//转发数据到business
	cl := &protocol2.ConnClose{}
	cl.UniqId = uniqId
	cl.Session = userSession
	router := &protocol2.Router{}
	router.Cmd = protocol2.Cmd_ConnClose
	router.Data, _ = proto.Marshal(cl)
	data, _ := proto.Marshal(router)
	worker.Send(data)
	//统计转发到business的次数与字节数
	metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(len(data) + 4)) //加上4字节，是因为tcp包头的缘故
}

func onMessage(conn *websocket.Conn, _ websocket.MessageType, data []byte) {
	//检查是否为心跳消息
	if bytes.Equal(data, heartbeat.PingMessage) {
		//响应客户端心跳
		err := conn.WriteMessage(websocket.TextMessage, heartbeat.PongMessage)
		if err != nil {
			_ = conn.Close()
		}
		metrics.Registry[metrics.ItemCustomerHeartbeat].Meter.Mark(1)
		return
	}
	//限制数据包大小
	if len(data)-3 > configs.Config.Customer.ReceivePackLimit {
		if err := conn.WriteMessage(websocket.TextMessage, dataTooLarge); err != nil {
			_ = conn.Close()
		}
		return
	}
	//读取前三个字节，转成business的服务编号
	workerId := utils.BytesToInt(data, 3)
	//限流检查
	if limit.Manager.Allow(workerId) == false {
		return
	}
	//从连接中拿出session
	session, ok := conn.Session().(*info.Info)
	if !ok {
		return
	}
	//获取能处理消息的business
	worker := workerManager.Manager.Get(workerId)
	if worker == nil {
		log.Logger.Warn().Int("workerId", workerId).Str("session", session.GetSession()).Bytes("customerToWorkerData", data[3:]).Msg("Not found business")
		return
	}
	//编码数据成business需要的格式
	tf := &protocol2.Transfer{}
	tf.Data = data[3:]
	session.GetToProtocolTransfer(tf)
	router := &protocol2.Router{}
	router.Cmd = protocol2.Cmd_Transfer
	router.Data, _ = proto.Marshal(tf)
	//转发数据到business
	data, _ = proto.Marshal(router)
	worker.Send(data)
	//统计转发到business的次数与字节数
	metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(len(data) + 4)) //加上4字节，是因为tcp包头的缘故
}

func checkOrigin(r *http.Request) bool {
	if len(configs.Config.Customer.AllowOrigin) == 0 {
		return true
	}
	origin := r.Header.Get("Origin")
	for _, v := range configs.Config.Customer.AllowOrigin {
		if strings.Contains(origin, v) {
			return true
		}
	}
	return false
}
