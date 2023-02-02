package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// Unsubscribe 取消订阅
func Unsubscribe(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.Unsubscribe{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.Unsubscribe error: %v", err)
		return
	}
	if payload.SessionId == 0 || len(payload.Topics) == 0 {
		return
	}
	conn := customerManager.Manager.Get(payload.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	//将自己的订阅信息移除掉
	info.Unsubscribe(payload.Topics)
	//主题管理里面也移除掉订阅关系
	session.Topics.Del(payload.Topics, payload.SessionId)
	//取消订阅后，有信息要传递给用户，则转发数据给到用户
	if len(payload.Data) > 0 {
		Catapult.Put(payload)
	}
}
