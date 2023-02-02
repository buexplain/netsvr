package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/session"
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
	bitmap := session.Topics.Get(payload.Topic)
	if bitmap == nil {
		return
	}
	peekAble := bitmap.Iterator()
	for peekAble.HasNext() {
		Catapult.Put(NewPayload(peekAble.Next(), payload.Data))
	}
}
