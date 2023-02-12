package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/catapult"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicUnsubscribe 取消订阅
func TopicUnsubscribe(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.TopicUnsubscribe{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.UnsubscribeTopics error: %v", err)
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
	topics := session.UnsubscribeTopics(payload.Topics)
	topic.Topic.Del(topics, payload.UniqId)
	if len(payload.Data) > 0 {
		catapult.Catapult.Put(payload)
	}
}
