package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/broadcast"
	workerManager "netsvr/internal/worker/manager"
)

// Broadcast 广播
func Broadcast(param []byte, _ *workerManager.ConnProcessor) {
	req := broadcast.Broadcast{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal broadcast.Broadcast error: %v", err)
		return
	}
	if len(req.Data) == 0 {
		return
	}
	bitmap := session.Id.GetAllocated()
	peekAble := bitmap.Iterator()
	for peekAble.HasNext() {
		Catapult.Put(NewPayload(peekAble.Next(), req.Data))
	}
}
