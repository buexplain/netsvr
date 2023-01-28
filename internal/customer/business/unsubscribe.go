package business

import (
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/unsubscribe"
)

// Unsubscribe 取消订阅
func Unsubscribe(unsubscribe *unsubscribe.Unsubscribe) {
	if unsubscribe.SessionId == 0 || len(unsubscribe.Topics) == 0 {
		return
	}
	conn := manager.Manager.Get(unsubscribe.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	//将自己的订阅信息移除掉
	info.Unsubscribe(unsubscribe.Topics)
	//主题管理里面也移除掉订阅关系
	session.Topics.Del(unsubscribe.Topics, unsubscribe.SessionId)
	//取消订阅后，有信息要传递给用户，则转发数据给到用户
	if len(unsubscribe.Data) > 0 {
		Catapult.Put(NewPayload(unsubscribe.SessionId, unsubscribe.Data))
	}
}
