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
	//更新用户的信息
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser != nil {
		user := userDb.Collect.GetUserById(currentUser.Id)
		if user != nil {
			userDb.Collect.SetOnline(user.Id, false)
		}
	}
	//删除网关信息
	ret := &internalProtocol.DeleteInfo{}
	ret.UniqId = tf.UniqId
	ret.DelUniqId = true
	ret.DelSession = true
	ret.DelTopic = true
	ret.Data = workerUtils.NewResponse(protocol.RouterLogout, map[string]interface{}{"code": 0, "message": "退出登录成功"})
	//回写给网关服务器
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_DeleteInfo
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
