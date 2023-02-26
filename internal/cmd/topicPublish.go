package cmd

import (
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicPublish 发布
func TopicPublish(param []byte, _ *workerManager.ConnProcessor) {
	payload := protocol.TopicPublish{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal publish.TopicPublish error: %v", err)
		return
	}
	if len(payload.Data) == 0 || payload.Topic == "" {
		return
	}
	uniqIds := topic.Topic.GetUniqIds(payload.Topic)
	if len(uniqIds) == 0 {
		return
	}
	for _, uniqId := range uniqIds {
		conn := manager.Manager.Get(uniqId)
		if conn == nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			_ = conn.Close()
		}
	}
}
