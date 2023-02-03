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
func Login(currentSessionId uint32, _ string, _ string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	login := new(protocol.Login)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), login); err != nil {
		logging.Error("Parse protocol.Login request error: %v", err)
		return
	}
	//构建一个发给网关的路由
	router := &internalProtocol.Router{}
	//要求网关设定登录状态
	router.Cmd = internalProtocol.Cmd_SetUserLoginStatus
	//构建一个包含登录状态相关是业务对象
	ret := &internalProtocol.SetUserLoginStatus{}
	ret.SessionId = currentSessionId
	//查找用户
	user := userDb.Collect.GetUser(login.Username)
	//校验账号密码，判断是否登录成功
	if user != nil && user.Password == login.Password {
		//更新用户的session id
		userDb.Collect.SetSessionId(user.Id, currentSessionId)
		ret.LoginStatus = true
		//响应给客户端的数据
		ret.Data = workerUtils.NewResponse(protocol.RouterLogin, map[string]interface{}{"code": 0, "message": "登录成功", "data": user.ToClientInfo()})
		//存储到网关的用户信息
		ret.UserInfo = user.ToNetSvrInfo()
		//这个id只在登录成功的时候设置
		ret.UserId = strconv.Itoa(user.Id)
		//初始化用户默认订阅的主题
		ret.Topics = user.Topics
	} else {
		ret.LoginStatus = false
		ret.Data = workerUtils.NewResponse(protocol.RouterLogin, map[string]interface{}{"code": 1, "message": "登录失败，账号或密码错误"})
	}
	//将业务对象放到路由上
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	//回写给网关服务器
	processor.Send(pt)
}
