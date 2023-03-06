package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
	"strings"
)

// SingleCast 单播
func SingleCast(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.SingleCast{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.SingleCast failed")
		return
	}
	if strings.EqualFold(payload.UniqId, "") || len(payload.Data) == 0 {
		return
	}
	conn := manager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
		_ = conn.Close()
	}
}
