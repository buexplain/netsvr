package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/userDb"
	workerUtils "netsvr/test/business/utils"
	"time"
)

// UpdateSessionUserInfo 更新网关中的用户信息
func UpdateSessionUserInfo(currentSessionId uint32, userInfo string, userId string, _ string, processor *connProcessor.ConnProcessor) {
	currentUser := userDb.ParseNetSvrInfo(userInfo)
	//这里就把时间改动一下，意思意思
	currentUser.LastUpdateTime = time.Now()
	msg := map[string]interface{}{"netSvrInfo": currentUser}
	ret := &internalProtocol.UpSessionUserInfo{}
	ret.SessionId = currentSessionId
	ret.UserId = userId
	ret.UserInfo = currentUser.Encode()
	ret.Data = workerUtils.NewResponse(protocol.RouterUpdateSessionUserInfo, map[string]interface{}{"code": 0, "message": "更新网关中的用户信息成功", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_UpSessionUserInfo
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
