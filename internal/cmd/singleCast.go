package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/catapult"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// SingleCast 单播
func SingleCast(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.SingleCast{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.SingleCast error: %v", err)
		return
	}
	if payload.UniqId != "" && len(payload.Data) > 0 {
		catapult.Catapult.Put(payload)
	}
}
