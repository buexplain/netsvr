package cmd

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/userDb"
	businessUtils "netsvr/test/business/utils"
	"strconv"
)

type sign struct{}

var Sign = sign{}

func (r sign) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterSignIn, r.SignIn)
	processor.RegisterBusinessCmd(protocol.RouterSignOut, r.SignOut)
}

// SignInParam 客户端发送的登录信息
type SignInParam struct {
	Username string
	Password string
}

// SignIn 登录
func (sign) SignIn(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	login := new(SignInParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), login); err != nil {
		logging.Error("Parse SignInParam error: %v", err)
		return
	}
	if login.Username == "" && login.Password == "" {
		return
	}
	ret := &internalProtocol.InfoUpdate{}
	ret.UniqId = tf.UniqId
	//查找用户
	user := userDb.Collect.GetUser(login.Username)
	//校验账号密码，判断是否登录成功
	if user == nil || user.Password != login.Password {
		ret.Data = businessUtils.NewResponse(protocol.RouterSignIn, map[string]interface{}{"code": 1, "message": "登录失败，账号或密码错误"})
	} else {
		if tf.Session == "" && user.IsOnline {
			//账号已经被登录了，通知已经登录者
			ret.DataAsNewUniqIdExisted = businessUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您的帐号在另一地点登录！"})
		}
		//设置当前登录信息
		userDb.Collect.SetOnline(user.Id, true)
		ret.NewUniqId = strconv.Itoa(user.Id)
		ret.NewSession = user.ToNetSvrInfo()
		ret.NewTopics = user.Topics
		ret.Data = businessUtils.NewResponse(protocol.RouterSignIn, map[string]interface{}{"code": 0, "message": "登录成功", "data": user.ToClientInfo()})
	}
	//回写给网关服务器
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_InfoUpdate
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// SignOut 退出登录
func (sign) SignOut(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
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
			ret.Data = businessUtils.NewResponse(protocol.RouterSignOut, map[string]interface{}{"code": 0, "message": "退出登录成功"})
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
	ret.Data = businessUtils.NewResponse(protocol.RouterSignOut, map[string]interface{}{"code": 1, "message": "您已经是退出登录状态！"})
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
