package customer

import (
	"bytes"
	"context"
	"github.com/antlabs/timer"
	"github.com/buexplain/netsvr/configs"
	"github.com/buexplain/netsvr/internal/customer/heartbeat"
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/customer/session"
	toWorkerRouter "github.com/buexplain/netsvr/internal/protocol/toWorker/router"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/transfer"
	workerManager "github.com/buexplain/netsvr/internal/worker/manager"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/buexplain/netsvr/pkg/utils"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"net/http"
	"time"
)

var server *nbhttp.Server

func Start() {
	mux := &http.ServeMux{}
	mux.HandleFunc("/gateway", onWebsocket)
	config := nbhttp.Config{
		Network: "tcp",
		Addrs:   []string{"localhost:8080"},
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
		info := &session.Info{}
		info.Id = session.Id.Get()
		conn.SetSession(info)
		manager.Manager.Set(info.Id, conn)
		logging.Debug("Customer websocket open, session: %#v", info)
	})
	upgrade.OnClose(func(conn *websocket.Conn, err error) {
		info, ok := conn.Session().(*session.Info)
		if ok {
			session.Id.Put(info.Id)
			manager.Manager.Del(info.Id)
		}
		logging.Debug("Customer websocket close: %v, session: %#v", err, info)
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
		//读取前三个字节，转成工作进程的id
		workerId := utils.BytesToInt(data, 3)
		//编码数据成工作进程需要的格式
		toWorkerRoute := &toWorkerRouter.Router{}
		toWorkerRoute.Cmd = toWorkerRouter.Cmd_Transfer
		tf := &transfer.Transfer{}
		tf.Data = data[3:]
		info, _ := conn.Session().(*session.Info)
		tf.SessionId = info.Id
		toWorkerRoute.Data, _ = proto.Marshal(tf)
		worker := workerManager.Manager.Get(workerId)
		if worker == nil {
			logging.Error("Not found worker by id: %d", workerId)
			return
		}
		//转发数据到工作进程
		data, _ = proto.Marshal(toWorkerRoute)
		_, _ = worker.Write(data)
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
	var heartbeatNode timer.TimeNoder
	heartbeatNode = heartbeat.Timer.ScheduleFunc(time.Duration(configs.Config.CustomerHeartbeatIntervalSecond)*time.Second, func() {
		//超过活跃期，服务端主动发送心跳
		if err := wsConn.WriteMessage(websocket.TextMessage, heartbeat.PingMessage); err != nil {
			if heartbeatNode != nil {
				heartbeatNode.Stop()
				heartbeatNode = nil
			}
			_ = conn.Close()
			logging.Error("Customer write error: %#v", err)
		}
	})
	logging.Debug("Customer websocket upgrade ok, remoteAddr: %s", conn.RemoteAddr())
}
