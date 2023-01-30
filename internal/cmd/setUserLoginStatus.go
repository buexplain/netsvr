package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/setUserLoginStatus"
	workerManager "netsvr/internal/worker/manager"
)

// SetUserLoginStatus 设置用户登录状态
func SetUserLoginStatus(param []byte, _ *workerManager.ConnProcessor) {
	req := setUserLoginStatus.SetUserLoginStatus{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal setUserLoginStatus.SetUserLoginStatus error: %v", err)
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
	//设置登录状态
	if req.LoginStatus {
		//登录成功，设置用户的信息
		info.SetLoginStatusOk(req.UserInfo)
	} else {
		//登录失败，重置登录状态为等待登录中，用户可以发起二次登录
		info.SetLoginStatus(session.LoginStatusWait)
	}
	//如果有消息要转告给用户，则转发消息给用户
	if len(req.Data) > 0 {
		Catapult.Put(NewPayload(req.SessionId, req.Data))
	}
}
