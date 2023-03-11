package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/protocol"
)

// TopicUnsubscribe 取消订阅
func TopicUnsubscribe(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.TopicUnsubscribe{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicUnsubscribe failed")
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
	session.MuxLock()
	currentUniqId := session.UnsubscribeTopics(payload.Topics)
	//这里根据session里面的uniqId去删除订阅关系，因为有可能当UnsubscribeTopics得到锁的时候，session里面的uniqId与当前的payload.UniqId不一致了
	topic.Topic.DelBySlice(payload.Topics, currentUniqId, payload.UniqId)
	session.MuxUnLock()
	if len(payload.Data) > 0 {
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			_ = conn.Close()
		}
	}
}
