package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/catapult"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	customerTopic "netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicDelete 删除主题
func TopicDelete(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.TopicDelete{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.TopicDelete error: %v", err)
		return
	}
	switch len(payload.Topics) {
	case 0:
		return
	case 1:
		uniqIds := customerTopic.Topic.Pull(payload.Topics[0], nil)
		if uniqIds == nil {
			return
		}
		for _, uniqId := range *uniqIds {
			conn := customerManager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			session, ok := conn.Session().(*info.Info)
			if !ok {
				continue
			}
			if session.UnsubscribeTopic(payload.Topics[0]) && len(payload.Data) > 0 {
				catapult.Catapult.Put(catapult.NewPayload(uniqId, payload.Data))
			}
		}
		uniqIds = nil
		return
	default:
		var uniqIds *[]string
		var send map[string]struct{}
		for _, topic := range payload.Topics {
			uniqIds = customerTopic.Topic.Pull(topic, uniqIds)
			if uniqIds == nil {
				continue
			}
			if send == nil && len(payload.Data) > 0 {
				send = make(map[string]struct{}, len(*uniqIds))
			}
			for _, uniqId := range *uniqIds {
				conn := customerManager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				session, ok := conn.Session().(*info.Info)
				if !ok {
					continue
				}
				if session.UnsubscribeTopic(topic) == false || send == nil {
					continue
				}
				if _, ok = send[uniqId]; !ok {
					catapult.Catapult.Put(catapult.NewPayload(uniqId, payload.Data))
					send[uniqId] = struct{}{}
				}
			}
		}
		uniqIds = nil
	}
}
