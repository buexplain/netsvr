package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// Publish 发布
func Publish(param []byte, _ *workerManager.ConnProcessor) {
	payload := protocol.Publish{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal publish.Publish error: %v", err)
		return
	}
	if len(payload.Data) == 0 || payload.Topic == "" {
		return
	}
	uniqIds := topic.Topic.Get(payload.Topic)
	if len(uniqIds) == 0 {
		return
	}
	for _, uuid := range uniqIds {
		Catapult.Put(NewPayload(uuid, payload.Data))
	}
	uniqIds = uniqIds[:0]
}
