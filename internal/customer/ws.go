package customer

import (
	"bytes"
	"context"
	"github.com/buexplain/netsvr/internal/customer/heartbeat"
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/internal/protocol/transferToWorker"
	workerManager "github.com/buexplain/netsvr/internal/worker/manager"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"net/http"
	"strconv"
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
		if messageType == websocket.PingMessage || bytes.Equal(data, heartbeat.PingMessage) {
			err := conn.WriteMessage(websocket.PongMessage, heartbeat.PongMessage)
			if err != nil {
				logging.Debug("Failed to send pong %#v", err)
				_ = conn.Close()
				return
			}
			return
		}
		//读取前三个字节，转成工作进程的id
		workerId, _ := strconv.Atoi(string(data[0:3]))
		//编码数据成工作进程需要的格式
		info, _ := conn.Session().(*session.Info)
		transfer := &transferToWorker.TransferToWorker{}
		transfer.Data = data[3:]
		transfer.SessionId = info.Id
		data, _ = proto.Marshal(transfer)
		worker := workerManager.Manager.Get(workerId)
		if worker == nil {
			logging.Error("Not found worker by id: %d", workerId)
			return
		}
		//转发数据到工作进程
		_, _ = worker.Write(data)
	})
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		logging.Error("Customer websocket upgrade failed: %v", err)
		return
	}
	select {
	case <-quit.Ctx.Done():
		_ = conn.Close()
	default:
		logging.Debug("Customer websocket upgrade ok, remoteAddr: %s", conn.RemoteAddr())
	}
}
