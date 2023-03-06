package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	customerTopic "netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicDelete 删除主题
func TopicDelete(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.TopicDelete{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicDelete failed")
		return
	}
	//topic --> []uniqId
	arr := customerTopic.Topic.PullAndReturnUniqIds(payload.Topics)
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
				if session, ok := conn.Session().(*info.Info); ok {
					_ = session.UnsubscribeTopic(topic)
				}
			}
		}
		return
	}
	//需要发送给客户数据，注意，只能发送一次
	isSend := false
	//先清空整个payload.Topics，用于记录已经处理过的topic
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
				//迭代每一个已经处理的topic，判断每个已经处理topic是否包含当前循环的uniqId
				if _, isSend = arr[topic][uniqId]; isSend {
					break
				}
			}
			//没有发送过数据，则发送数据
			if !isSend {
				if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
					_ = conn.Close()
				}
			}
		}
		//处理完毕的topic，记录起来
		payload.Topics = append(payload.Topics, topic)
	}
}
