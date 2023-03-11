package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/protocol"
)

// Broadcast 广播
func Broadcast(param []byte, _ *workerManager.ConnProcessor) {
	payload := protocol.Broadcast{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.Broadcast failed")
		return
	}
	if len(payload.Data) == 0 {
		return
	}
	//取出所有的连接
	connections := make([]*websocket.Conn, 0, customerManager.Manager.Len())
	for _, c := range customerManager.Manager {
		c.GetConnections(&connections)
	}
	//循环所有的连接，挨个发送出去
	for _, conn := range connections {
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			_ = conn.Close()
		}
	}
}
