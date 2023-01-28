package business

import (
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/setSessionUser"
)

// SetSessionUser 设置网关的session面存储的用户信息，只有登录成功的才会生效，因为没登录的，在后续登录成功后也会设置一次
func SetSessionUser(setSessionUser *setSessionUser.SetSessionUser) {
	conn := manager.Manager.Get(setSessionUser.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	if !info.SetUserOnLoginStatusOk(setSessionUser.UserInfo) {
		//设置失败，直接返回，不会将用户数据转发给用户
		return
	}
	if len(setSessionUser.Data) > 0 {
		Catapult.Put(NewPayload(setSessionUser.SessionId, setSessionUser.Data))
	}
}
