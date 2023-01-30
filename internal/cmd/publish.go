package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/publish"
	workerManager "netsvr/internal/worker/manager"
)

// Publish 发布
func Publish(param []byte, _ *workerManager.ConnProcessor) {
	req := publish.Publish{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal publish.Publish error: %v", err)
		return
	}
	if len(req.Data) == 0 || req.Topic == "" {
		return
	}
	bitmap := session.Topics.Get(req.Topic)
	if bitmap == nil {
		return
	}
	peekAble := bitmap.Iterator()
	for peekAble.HasNext() {
		Catapult.Put(NewPayload(peekAble.Next(), req.Data))
	}
}
