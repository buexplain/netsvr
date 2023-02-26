package cmd

import (
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/manager"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// Multicast 组播
func Multicast(param []byte, _ *workerManager.ConnProcessor) {
	payload := protocol.Multicast{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.Multicast error: %v", err)
		return
	}
	if len(payload.Data) == 0 {
		return
	}
	for _, uniqId := range payload.UniqIds {
		conn := manager.Manager.Get(uniqId)
		if conn == nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			_ = conn.Close()
		}
	}
}
