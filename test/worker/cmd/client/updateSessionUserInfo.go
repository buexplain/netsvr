package client

import (
	"google.golang.org/protobuf/proto"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/setSessionUser"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/userDb"
	workerUtils "netsvr/test/worker/utils"
	"time"
)

// UpdateSessionUserInfo 更新网关中的用户信息
func UpdateSessionUserInfo(currentSessionId uint32, userStr string, _ string, processor *connProcessor.ConnProcessor) {
	currentUser := userDb.ParseNetSvrInfo(userStr)
	currentUser.LastUpdateTime = time.Now()
	msg := map[string]interface{}{"netSvrInfo": currentUser}
	ret := &setSessionUser.SetSessionUser{}
	ret.SessionId = currentSessionId
	ret.UserInfo = currentUser.Encode()
	ret.Data = workerUtils.NewResponse(protocol.RouterUpdateSessionUserInfo, map[string]interface{}{"code": 0, "message": "更新网关中的用户信息成功", "data": msg})
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SetSessionUser
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
