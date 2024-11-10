/**
* Copyright 2023 buexplain@qq.com
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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"netsvr/test/business/configs"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
	"strconv"
	"strings"
	"sync/atomic"
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

// SignInForForgeParam 伪造登录时，需要初始化的数据
type SignInForForgeParam struct {
	TopicNum   uint `json:"topicNum"`
	SessionLen uint `json:"sessionLen"`
}

// 伪造登录时，伪造客户业务系统的唯一id的初始值
var forgeCustomerId uint32

func init() {
	forgeCustomerId = configs.Config.ForgeCustomerIdInitVal
}

func (sign) SignInForForge(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(SignInForForgeParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse SignInForForgeParam failed")
		return
	}
	ret := &netsvrProtocol.ConnInfoUpdate{}
	ret.UniqId = tf.UniqId
	//伪造主题
	if payload.TopicNum > 0 && payload.TopicNum <= 10000 {
		ret.NewTopics = make([]string, 0, payload.TopicNum)
		for i := 0; i < int(payload.TopicNum); i++ {
			ret.NewTopics = append(ret.NewTopics, testUtils.GetRandStr(2))
		}
	}
	//伪造客户id
	customerId := atomic.AddUint32(&forgeCustomerId, 1)
	ret.NewCustomerId = strconv.FormatInt(int64(customerId), 10)
	//伪造的session值
	if payload.SessionLen > 0 && payload.SessionLen <= 1024*1024 {
		ret.NewSession = strings.Repeat("s", int(payload.SessionLen))
	}
	ret.Data = testUtils.NewResponse(protocol.RouterSignInForForge, map[string]interface{}{"code": 0, "message": "登录成功", "data": userDb.ClientInfo{
		Id:     customerId,
		Name:   ret.NewCustomerId,
		UniqId: tf.UniqId,
	}})
	processor.Send(ret, netsvrProtocol.Cmd_ConnInfoUpdate)
}

func (sign) SignOutForForge(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	//删除网关信息
	ret := &netsvrProtocol.ConnInfoDelete{}
	ret.UniqId = tf.UniqId
	ret.DelCustomerId = true
	ret.DelSession = true
	ret.DelTopic = true
	ret.Data = testUtils.NewResponse(protocol.RouterSignOutForForge, map[string]interface{}{"code": 0, "message": "退出登录成功"})
	processor.Send(ret, netsvrProtocol.Cmd_ConnInfoDelete)
}

// SignIn 登录
func (sign) SignIn(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	login := new(SignInParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), login); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse SignInParam failed")
		return
	}
	//校验参数
	if login.Username == "" || login.Password == "" {
		invalidArgument := &netsvrProtocol.SingleCast{}
		invalidArgument.UniqId = tf.UniqId
		invalidArgument.Data = testUtils.NewResponse(protocol.RouterSignIn, map[string]interface{}{"code": 1, "message": "请输入账号、密码", "data": nil})
		processor.Send(invalidArgument, netsvrProtocol.Cmd_SingleCast)
		return
	}
	//查找用户
	user := userDb.Collect.GetUser(login.Username)
	//校验账号密码，判断是否登录成功
	if user == nil || user.Password != login.Password {
		invalidArgument := &netsvrProtocol.SingleCast{}
		invalidArgument.UniqId = tf.UniqId
		invalidArgument.Data = testUtils.NewResponse(protocol.RouterSignIn, map[string]interface{}{"code": 1, "message": "登录失败，账号或密码错误", "data": nil})
		processor.Send(invalidArgument, netsvrProtocol.Cmd_SingleCast)
		return
	}
	//将当前登录信息存储到网关
	ret := &netsvrProtocol.ConnInfoUpdate{}
	userDb.Collect.SetOnlineInfo(user.Id, tf.UniqId)
	ret.UniqId = tf.UniqId
	ret.NewCustomerId = strconv.FormatInt(int64(user.Id), 10)
	ret.NewSession = user.ToNetSvrInfo()
	ret.NewTopics = user.Topics
	ret.Data = testUtils.NewResponse(protocol.RouterSignIn, map[string]interface{}{"code": 0, "message": "登录成功", "data": user.ToClientInfo()})
	processor.Send(ret, netsvrProtocol.Cmd_ConnInfoUpdate)
}

// SignOut 退出登录
func (sign) SignOut(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	//删除网关信息
	ret := &netsvrProtocol.ConnInfoDelete{}
	ret.UniqId = tf.UniqId
	ret.DelCustomerId = true
	ret.DelSession = true
	ret.DelTopic = true
	ret.Data = testUtils.NewResponse(protocol.RouterSignOut, map[string]interface{}{"code": 0, "message": "退出登录成功"})
	processor.Send(ret, netsvrProtocol.Cmd_ConnInfoDelete)
	//删除数据库信息
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser != nil {
		user := userDb.Collect.GetUserById(currentUser.Id)
		if user != nil {
			userDb.Collect.SetOnlineInfo(user.Id, "")
		}
	}
}
