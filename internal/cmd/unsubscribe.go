package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/unsubscribe"
	workerManager "netsvr/internal/worker/manager"
)

// Unsubscribe 取消订阅
func Unsubscribe(param []byte, _ *workerManager.ConnProcessor) {
	req := unsubscribe.Unsubscribe{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal unsubscribe.Unsubscribe error: %v", err)
		return
	}
	if req.SessionId == 0 || len(req.Topics) == 0 {
		return
	}
	conn := manager.Manager.Get(req.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	//将自己的订阅信息移除掉
	info.Unsubscribe(req.Topics)
	//主题管理里面也移除掉订阅关系
	session.Topics.Del(req.Topics, req.SessionId)
	//取消订阅后，有信息要传递给用户，则转发数据给到用户
	if len(req.Data) > 0 {
		Catapult.Put(NewPayload(req.SessionId, req.Data))
	}
}
