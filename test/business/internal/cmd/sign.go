/**
* Copyright 2022 buexplain@qq.com
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package cmd

import (
	"encoding/json"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/protocol"
	"google.golang.org/protobuf/proto"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
	"strings"
)

type sign struct{}

var Sign = sign{}

func (r sign) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterSignIn, r.SignIn)
	processor.RegisterBusinessCmd(protocol.RouterSignOut, r.SignOut)
	processor.RegisterBusinessCmd(protocol.RouterSignInForForge, r.SignInForForge)
	processor.RegisterBusinessCmd(protocol.RouterSignOutForForge, r.SignOutForForge)
}

// SignInParam 客户端发送的登录信息
type SignInParam struct {
	Username string
	Password string
}

// 一个伪造的session值
var forgeSession = strings.Repeat("s", 1024)

func (sign) SignInForForge(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	ret := &netsvrProtocol.InfoUpdate{}
	ret.UniqId = tf.UniqId
	//伪造uniqId
	if len(tf.UniqId) >= 18 {
		ret.NewUniqId = tf.UniqId[0:18] + testUtils.GetRandStr(2)
	} else {
		ret.NewUniqId = tf.UniqId + testUtils.GetRandStr(2)
	}
	//伪造主题
	ret.NewTopics = make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		ret.NewTopics = append(ret.NewTopics, testUtils.GetRandStr(2))
	}
	//伪造的session值，存粹是为了模拟正常业务的数据大小
	ret.NewSession = forgeSession
	ret.Data = testUtils.NewResponse(protocol.RouterSignInForForge, map[string]interface{}{"code": 0, "message": "登录成功", "data": userDb.ClientInfo{UniqId: ret.NewUniqId}})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_InfoUpdate
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

func (sign) SignOutForForge(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	//删除网关信息
	ret := &netsvrProtocol.InfoDelete{}
	ret.UniqId = tf.UniqId
	ret.DelUniqId = true
	ret.DelSession = true
	ret.DelTopic = true
	ret.Data = testUtils.NewResponse(protocol.RouterSignOutForForge, map[string]interface{}{"code": 0, "message": "退出登录成功"})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_InfoDelete
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// SignIn 登录
func (sign) SignIn(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	login := new(SignInParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), login); err != nil {
		log.Logger.Error().Err(err).Msg("Parse SignInParam failed")
		return
	}
	if login.Username == "" && login.Password == "" {
		return
	}
	ret := &netsvrProtocol.InfoUpdate{}
	ret.UniqId = tf.UniqId
	//查找用户
	user := userDb.Collect.GetUser(login.Username)
	//校验账号密码，判断是否登录成功
	if user == nil || user.Password != login.Password {
		ret.Data = testUtils.NewResponse(protocol.RouterSignIn, map[string]interface{}{"code": 1, "message": "登录失败，账号或密码错误"})
	} else {
		if tf.Session == "" && user.IsOnline {
			//账号已经被登录了，通知已经登录者
			ret.DataAsNewUniqIdExisted = testUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您的帐号在另一地点登录！"})
		}
		//设置当前登录信息
		data := user.ToClientInfo()
		userDb.Collect.SetOnline(user.Id, true)
		ret.NewUniqId = data.UniqId
		ret.NewSession = user.ToNetSvrInfo()
		ret.NewTopics = user.Topics
		ret.Data = testUtils.NewResponse(protocol.RouterSignIn, map[string]interface{}{"code": 0, "message": "登录成功", "data": data})
	}
	//回写给网关服务器
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_InfoUpdate
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// SignOut 退出登录
func (sign) SignOut(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	router := &netsvrProtocol.Router{}
	if currentUser != nil {
		user := userDb.Collect.GetUserById(currentUser.Id)
		if user != nil {
			userDb.Collect.SetOnline(user.Id, false)
			//删除网关信息
			ret := &netsvrProtocol.InfoDelete{}
			ret.UniqId = tf.UniqId
			ret.DelUniqId = true
			ret.DelSession = true
			ret.DelTopic = true
			ret.Data = testUtils.NewResponse(protocol.RouterSignOut, map[string]interface{}{"code": 0, "message": "退出登录成功"})
			router.Cmd = netsvrProtocol.Cmd_InfoDelete
			router.Data, _ = proto.Marshal(ret)
			pt, _ := proto.Marshal(router)
			processor.Send(pt)
			return
		}
	}
	//还未登录
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	ret.Data = testUtils.NewResponse(protocol.RouterSignOut, map[string]interface{}{"code": 1, "message": "您已经是退出登录状态！"})
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
