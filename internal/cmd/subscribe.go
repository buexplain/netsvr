package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// Subscribe 订阅
func Subscribe(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.Subscribe{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.Subscribe error: %v", err)
		return
	}
	if payload.UniqId == "" || len(payload.Topics) == 0 {
		return
	}
	conn := customerManager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	session, ok := conn.Session().(*info.Info)
	if !ok {
		return
	}
	topics := session.Subscribe(payload.Topics)
	topic.Topic.Set(topics, payload.UniqId)
	if len(payload.Data) > 0 {
		Catapult.Put(payload)
	}
}
