package client

import (
	"google.golang.org/protobuf/proto"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/setUserLoginStatus"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/userDb"
	workerUtils "netsvr/test/worker/utils"
)

// Logout 退出登录
func Logout(currentSessionId uint32, userStr string, _ string, processor *connProcessor.ConnProcessor) {
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//要求网关设定登录状态
	toServerRoute.Cmd = toServerRouter.Cmd_SetUserLoginStatus
	//构建一个包含登录状态相关是业务对象
	ret := &setUserLoginStatus.SetUserLoginStatus{}
	ret.SessionId = currentSessionId
	ret.LoginStatus = false
	ret.Data = workerUtils.NewResponse(protocol.RouterLogout, map[string]interface{}{"code": 0, "message": "退出登录成功"})
	//更新用户的信息
	currentUser := userDb.ParseNetSvrInfo(userStr)
	if currentUser != nil {
		user := userDb.Collect.GetUser(currentUser.Name)
		if user != nil {
			//更新用户的session id
			userDb.Collect.SetSessionId(user.Id, 0)
		}
	}
	//将业务对象放到路由上
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	//回写给网关服务器
	processor.Send(pt)
}
