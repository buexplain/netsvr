package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/setSessionUser"
	workerManager "netsvr/internal/worker/manager"
)

// SetSessionUser 设置网关的session面存储的用户信息，只有登录成功的才会生效，因为没登录的，在后续登录成功后也会设置一次
func SetSessionUser(param []byte, _ *workerManager.ConnProcessor) {
	req := setSessionUser.SetSessionUser{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal setSessionUser.SetSessionUser error: %v", err)
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
	if !info.SetUserOnLoginStatusOk(req.UserInfo) {
		//设置失败，直接返回，不会将用户数据转发给用户
		return
	}
	if len(req.Data) > 0 {
		Catapult.Put(NewPayload(req.SessionId, req.Data))
	}
}
