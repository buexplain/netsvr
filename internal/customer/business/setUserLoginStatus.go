package business

import (
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/internal/protocol/toServer/setUserLoginStatus"
)

// SetUserLoginStatus 设置用户登录状态
func SetUserLoginStatus(userLoginStatus *setUserLoginStatus.SetUserLoginStatus) {
	conn := manager.Manager.Get(userLoginStatus.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	//设置登录状态
	if userLoginStatus.LoginStatus {
		//登录成功，设置用户的信息
		info.SetLoginStatusOk(userLoginStatus.UserInfo)
	} else {
		//登录失败，重置登录状态为等待登录中，用户可以发起二次登录
		info.SetLoginStatus(session.LoginStatusWait)
	}
	//如果有消息要转告给用户，则转发消息给用户
	if len(userLoginStatus.Data) > 0 {
		Catapult.Put(NewPayload(userLoginStatus.SessionId, userLoginStatus.Data))
	}
}
