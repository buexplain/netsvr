package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
	"strings"
)

// TopicPublish 发布
func TopicPublish(param []byte, _ *workerManager.ConnProcessor) {
	payload := protocol.TopicPublish{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicPublish failed")
		return
	}
	if len(payload.Data) == 0 || strings.EqualFold(payload.Topic, "") {
		return
	}
	uniqIds := topic.Topic.GetUniqIds(payload.Topic)
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
