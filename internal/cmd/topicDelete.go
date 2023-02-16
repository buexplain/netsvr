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
	arr := customerTopic.Topic.Pull(payload.Topics)
	if len(arr) == 0 {
		return
	}
	//只做删除，不需要通知到客户
	if len(payload.Data) == 0 {
		for topic, uniqIds := range arr {
			for uniqId := range uniqIds {
				conn := customerManager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				session, ok := conn.Session().(*info.Info)
				if !ok {
					continue
				}
				session.UnsubscribeTopic(topic)
			}
		}
		return
	}
	//需要发送给客户数据，注意，只能发送一次
	isSend := false
	payload.Topics = payload.Topics[:0]
	for topic, uniqIds := range arr {
		for uniqId := range uniqIds {
			conn := customerManager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			session, ok := conn.Session().(*info.Info)
			if !ok {
				continue
			}
			if !session.UnsubscribeTopic(topic) {
				continue
			}
			//取消订阅成功，判断是否已经发送过数据
			isSend = false
			for _, topic = range payload.Topics {
				if _, isSend = arr[topic][uniqId]; isSend {
					break
				}
			}
			//没有发送过数据，则发送数据
			if !isSend {
				catapult.Catapult.Put(catapult.NewPayload(uniqId, payload.Data))
			}
		}
		payload.Topics = append(payload.Topics, topic)
	}
}
