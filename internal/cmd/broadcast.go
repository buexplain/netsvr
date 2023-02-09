package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// Broadcast 广播
func Broadcast(param []byte, _ *workerManager.ConnProcessor) {
	payload := protocol.Broadcast{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.Broadcast error: %v", err)
		return
	}
	if len(payload.Data) == 0 {
		return
	}
	uniqIds := make([]string, 0, customerManager.Manager[0].Len())
	for _, c := range customerManager.Manager {
		c.GetUniqIds(&uniqIds)
		for _, uuid := range uniqIds {
			Catapult.Put(NewPayload(uuid, payload.Data))
		}
		uniqIds = uniqIds[:0]
	}
}
