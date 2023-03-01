package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicSubscribe 订阅
func TopicSubscribe(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.TopicSubscribe{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicSubscribe failed")
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
	var topics []string
	topics, payload.UniqId = session.SubscribeTopics(payload.Topics, true)
	//这里根据session里面的uniqId去构建订阅关系，因为有可能当SubscribeTopics得到锁的时候，session里面的uniqId与当前的payload.UniqId不一致了
	topic.Topic.Set(topics, payload.UniqId)
	if len(payload.Data) > 0 {
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			_ = conn.Close()
		}
	}
}
