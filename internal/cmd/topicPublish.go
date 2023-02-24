package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/catapult"
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
	for _, uuid := range uniqIds {
		catapult.Catapult.Put(catapult.NewPayload(uuid, payload.Data))
	}
	uniqIds = uniqIds[:0]
}