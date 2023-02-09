package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/userDb"
	workerUtils "netsvr/test/business/utils"
	"strconv"
)

// Login 登录
func Login(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	login := new(protocol.Login)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), login); err != nil {
		logging.Error("Parse protocol.Login request error: %v", err)
		return
	}
	if login.Username == "" && login.Password == "" {
		return
	}
	ret := &internalProtocol.UpdateInfo{}
	ret.UniqId = tf.UniqId
	//查找用户
	user := userDb.Collect.GetUser(login.Username)
	//校验账号密码，判断是否登录成功
	if user != nil && user.Password == login.Password {
		if user.IsOnline {
			//已经登录
			ret.Data = workerUtils.NewResponse(protocol.RouterLogin, map[string]interface{}{"code": 1, "message": "登录失败，您的帐号在另一地点登录！"})
		} else {
			user.IsOnline = true
			ret.NewUniqId = strconv.Itoa(user.Id)
			ret.NewSession = user.ToNetSvrInfo()
			ret.NewTopics = user.Topics
			ret.Data = workerUtils.NewResponse(protocol.RouterLogin, map[string]interface{}{"code": 0, "message": "登录成功", "data": user.ToClientInfo()})
		}
	} else {
		ret.Data = workerUtils.NewResponse(protocol.RouterLogin, map[string]interface{}{"code": 1, "message": "登录失败，账号或密码错误"})
	}
	//回写给网关服务器
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_UpdateInfo
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
