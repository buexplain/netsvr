package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/userDb"
	workerUtils "netsvr/test/business/utils"
)

// Logout 退出登录
func Logout(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	router := &internalProtocol.Router{}
	if currentUser != nil {
		user := userDb.Collect.GetUserById(currentUser.Id)
		if user != nil {
			userDb.Collect.SetOnline(user.Id, false)
			//删除网关信息
			ret := &internalProtocol.InfoDelete{}
			ret.UniqId = tf.UniqId
			ret.DelUniqId = true
			ret.DelSession = true
			ret.DelTopic = true
			ret.Data = workerUtils.NewResponse(protocol.RouterLogout, map[string]interface{}{"code": 0, "message": "退出登录成功"})
			router.Cmd = internalProtocol.Cmd_InfoDelete
			router.Data, _ = proto.Marshal(ret)
			pt, _ := proto.Marshal(router)
			processor.Send(pt)
			return
		}
	}
	//还未登录
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	ret.Data = workerUtils.NewResponse(protocol.RouterLogout, map[string]interface{}{"code": 1, "message": "您已经是退出登录状态！"})
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
