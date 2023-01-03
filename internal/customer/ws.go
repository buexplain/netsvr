package customer

import (
	"context"
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"net/http"
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
	upgrade.OnMessage(func(conn *websocket.Conn, messageType websocket.MessageType, bytes []byte) {
		_ = conn.WriteMessage(messageType, bytes)
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
