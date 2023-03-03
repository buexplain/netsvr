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
	"netsvr/internal/heartbeat"
	"netsvr/internal/limit"
	"netsvr/internal/log"
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
	mux.HandleFunc(configs.Config.Customer.HandlePattern, onWebsocket)
	config := nbhttp.Config{
		Network: "tcp",
		Addrs:   []string{configs.Config.Customer.ListenAddress},
		Handler: mux,
		MaxLoad: configs.Config.Customer.MaxOnlineNum,
		//TODO 测试更多的配置信息
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
		log.Logger.Error().Err(err).Msg("Customer websocket shutdown failed")
		return
	}
	log.Logger.Info().Msg("Customer websocket shutdown")
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
	upgrade.KeepaliveTime = configs.Config.Customer.ReadDeadline
	upgrade.CheckOrigin = checkOrigin
	upgrade.OnOpen(onOpen)
	upgrade.OnClose(onClose)
	upgrade.OnMessage(onMessage)
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Customer websocket upgrade failed")
		return
	}
	log.Logger.Debug().Interface("remoteAddr", conn.RemoteAddr()).Msg("Customer websocket upgrade ok")
}

func onOpen(conn *websocket.Conn) {
	//获取处理连接打开的worker
	workerId := workerManager.GetProcessConnOpenWorkerId()
	//限流检查，之所以要判断workerId大于0，是因为有可能业务方并不关心连接的打开信息，连接打开信息不会传递到业务方，则不必限流
	if workerId > 0 && limit.Manager.Allow(workerId) == false {
		_ = conn.Close()
		return
	}
	//统计打开连接次数
	metrics.Registry[metrics.ItemCustomerConnOpen].Meter.Mark(1)
	//分配uniqId，并将添加到管理器中
	uniqId := utils.UniqId()
	session := info.NewInfo(uniqId)
	conn.SetSession(session)
	manager.Manager.Set(uniqId, conn)
	log.Logger.Debug().Str("uniqId", uniqId).Msg("Customer websocket open")
	//获取能够处理连接打开信息的business
	worker := workerManager.Manager.Get(workerId)
	if worker == nil {
		log.Logger.Debug().Int("workerId", workerId).Msg("Not found process conn open business")
		return
	}
	//连接打开消息回传给business
	co := &protocol.ConnOpen{}
	co.UniqId = uniqId
	router := &protocol.Router{}
	router.Cmd = protocol.Cmd_ConnOpen
	router.Data, _ = proto.Marshal(co)
	data, _ := proto.Marshal(router)
	worker.Send(data)
	//统计转发到business的次数与字节数
	metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(len(data) + 4)) //加上4字节，是因为tcp包头的缘故
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
	//断开info与conn的关系
	conn.SetSession(nil)
	//清空info，并返回相关数据
	topics, uniqId, userSession := session.Clear(false)
	if uniqId == "" {
		session.MuxUnLock()
		//当前连接已经被清空了uniqId（可能被强制踢下线或者是被uniqId被顶掉了），无需进行接下来的逻辑
		return
	}
	//从连接管理器中删除
	manager.Manager.Del(uniqId)
	//删除订阅关系
	topic.Topic.Del(topics, uniqId, "")
	//释放锁
	session.MuxUnLock()
	log.Logger.Debug().Strs("topics", topics).Str("uniqId", uniqId).Str("session", userSession).Msg("Customer websocket close")
	//连接关闭消息回传给business
	workerId := workerManager.GetProcessConnCloseWorkerId()
	worker := workerManager.Manager.Get(workerId)
	if worker == nil {
		log.Logger.Debug().Int("workerId", workerId).Msg("Not found process conn close business")
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
}

func onMessage(conn *websocket.Conn, messageType websocket.MessageType, data []byte) {
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
