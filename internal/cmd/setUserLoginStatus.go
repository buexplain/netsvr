package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// SetUserLoginStatus 设置用户登录状态
func SetUserLoginStatus(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.SetUserLoginStatus{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.SetUserLoginStatus error: %v", err)
		return
	}
	conn := manager.Manager.Get(payload.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	//设置登录状态
	if payload.LoginStatus {
		//登录成功，设置用户的信息
		info.SetLoginStatusOk(payload.UserInfo, payload.UserId)
		//登录成功，先清空所有上一个账号订阅的主题
		session.Topics.Del(info.PullTopics(), payload.SessionId)
		//再订阅本次登录需要的主题
		info.Subscribe(payload.Topics)
		session.Topics.Set(payload.Topics, payload.SessionId)
	} else {
		//登录失败，重置登录状态为等待登录中，用户可以发起二次登录
		info.SetLoginStatusWait()
		//清空登录成功时候订阅的各种主题
		session.Topics.Del(info.PullTopics(), payload.SessionId)
	}
	//如果有消息要转告给用户，则转发消息给用户
	if len(payload.Data) > 0 {
		Catapult.Put(payload)
	}
}
