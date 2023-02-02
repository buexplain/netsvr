package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// SingleCast 单播
func SingleCast(param []byte, _ *workerManager.ConnProcessor) {
	req := protocol.SingleCast{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal protocol.SingleCast error: %v", err)
		return
	}
	if req.SessionId > 0 && len(req.Data) > 0 {
		Catapult.Put(NewPayload(req.SessionId, req.Data))
	}
}
