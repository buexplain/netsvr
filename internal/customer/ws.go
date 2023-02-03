package customer

import (
	"bytes"
	"context"
	"github.com/antlabs/timer"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"net/http"
	"netsvr/configs"
	"netsvr/internal/customer/heartbeat"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/quit"
	"netsvr/pkg/timecache"
	"netsvr/pkg/utils"
	"time"
)

var server *nbhttp.Server

func Start() {
	mux := &http.ServeMux{}
	mux.HandleFunc(configs.Config.CustomerHandlePattern, onWebsocket)
	config := nbhttp.Config{
		Network: "tcp",
		Addrs:   []string{configs.Config.CustomerListenAddress},
		Handler: mux,
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
	upgrade.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	upgrade.OnOpen(func(conn *websocket.Conn) {
		//分配session id，并将添加到管理器中
		info := session.NewInfo(session.Id.Pull())
		conn.SetSession(info)
		manager.Manager.Set(info.GetSessionId(), conn)
		//连接打开消息回传给business
		workerId := workerManager.GetProcessConnOpenWorkerId()
		worker := workerManager.Manager.Get(workerId)
		if worker == nil {
			logging.Debug("Not found process conn open business by id: %d", workerId)
			return
		}
		router := &protocol.Router{}
		router.Cmd = protocol.Cmd_ConnOpen
		co := &protocol.ConnOpen{}
		co.SessionId = info.GetSessionId()
		router.Data, _ = proto.Marshal(co)
		//转发数据到business
		data, _ := proto.Marshal(router)
		worker.Send(data)
		logging.Debug("Customer websocket open, session: %#v", info)
	})
	upgrade.OnClose(func(conn *websocket.Conn, err error) {
		info, ok := conn.Session().(*session.Info)
		if !ok {
			return
		}
		logging.Debug("Customer websocket close, session: %#v", info)
		//先在连接管理中剔除该连接
		manager.Manager.Del(info.GetSessionId())
		//回收session id
		session.Id.Put(info.GetSessionId())
		//取消订阅管理中，它的session id的任何关联
		session.Topics.Del(info.PullTopics(), info.GetSessionId())
		//连接关闭消息回传给business
		workerId := workerManager.GetProcessConnCloseWorkerId()
		worker := workerManager.Manager.Get(workerId)
		if worker == nil {
			logging.Debug("Not found process conn close business by id: %d", workerId)
			return
		}
		router := &protocol.Router{}
		router.Cmd = protocol.Cmd_ConnClose
		cls := &protocol.ConnClose{}
		info.GetToConnCloseObj(cls)
		router.Data, _ = proto.Marshal(cls)
		//转发数据到business
		data, _ := proto.Marshal(router)
		worker.Send(data)
	})
	upgrade.OnMessage(func(conn *websocket.Conn, messageType websocket.MessageType, data []byte) {
		//检查是否为心跳消息
		if bytes.Equal(data, heartbeat.PingMessage) {
			//响应客户端心跳
			err := conn.WriteMessage(websocket.TextMessage, heartbeat.PongMessage)
			if err != nil {
				_ = conn.Close()
			}
			return
		} else if bytes.Equal(data, heartbeat.PongMessage) {
			//客户端响应了服务端的心跳
			return
		} else if messageType == websocket.PingMessage {
			//响应客户端心跳
			err := conn.WriteMessage(websocket.PongMessage, heartbeat.PongMessage)
			if err != nil {
				_ = conn.Close()
			}
			return
		}
		info, ok := conn.Session().(*session.Info)
		if !ok {
			return
		}
		//更新连接的最后活跃时间
		info.SetLastActiveTime(timecache.Unix())
		//判断登录状态
		loginStatus := info.GetLoginStatus()
		if loginStatus == session.LoginStatusOk {
			goto label
		} else if loginStatus == session.LoginStatusWait {
			//等待登录，允许客户端发送数据到business，进行登录操作
			info.SetLoginStatusIng()
			goto label
		} else if loginStatus == session.LoginStatusIng {
			//登录中，不允许客户端发送任何数据
			return
		}
	label:
		//读取前三个字节，转成business的服务编号
		workerId := utils.BytesToInt(data, 3)
		//编码数据成business需要的格式
		router := &protocol.Router{}
		router.Cmd = protocol.Cmd_Transfer
		tf := &protocol.Transfer{}
		tf.Data = data[3:]
		info.GetToTransferObj(tf)
		router.Data, _ = proto.Marshal(tf)
		worker := workerManager.Manager.Get(workerId)
		if worker == nil {
			logging.Debug("Not found business by id: %d", workerId)
			return
		}
		//转发数据到business
		data, _ = proto.Marshal(router)
		worker.Send(data)
	})
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		logging.Error("Customer websocket upgrade failed: %v", err)
		return
	}
	wsConn := conn.(*websocket.Conn)
	if err := wsConn.SetReadDeadline(time.Time{}); err != nil {
		logging.Error("Customer websocket SetReadDeadline failed: %v", err)
		return
	}
	//设置心跳
	var heartbeatNode timer.TimeNoder
	heartbeatNode = heartbeat.Timer.ScheduleFunc(time.Duration(configs.Config.CustomerHeartbeatIntervalSecond)*time.Second, func() {
		info, ok := wsConn.Session().(*session.Info)
		if !ok {
			return
		}
		if timecache.Unix()-info.GetLastActiveTime() < configs.Config.CustomerHeartbeatIntervalSecond {
			//还在活跃期内，不做处理
			return
		}
		//超过活跃期，服务端主动发送心跳
		if err := wsConn.WriteMessage(websocket.TextMessage, heartbeat.PingMessage); err == nil {
			//写入数据成功，更新连接的最后活跃时间
			info.SetLastActiveTime(timecache.Unix())
			return
		}
		//写入数据失败，关闭心跳
		if heartbeatNode != nil {
			heartbeatNode.Stop()
			heartbeatNode = nil
		}
		//关闭连接
		_ = conn.Close()
	})
	logging.Debug("Customer websocket upgrade ok, remoteAddr: %s", conn.RemoteAddr())
}
