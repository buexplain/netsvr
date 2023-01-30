package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol/toServer/multicast"
	workerManager "netsvr/internal/worker/manager"
)

// Multicast 组播
func Multicast(param []byte, _ *workerManager.ConnProcessor) {
	req := multicast.Multicast{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal multicast.Multicast error: %v", err)
		return
	}
	if len(req.Data) == 0 {
		return
	}
	for _, sessionId := range req.SessionIds {
		Catapult.Put(NewPayload(sessionId, req.Data))
	}
}
