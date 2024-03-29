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
	"fmt"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v2/netsvr"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
	"strconv"
)

type multicast struct{}

var Multicast = multicast{}

func (r multicast) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterMulticastForUserId, r.ForUserId)
	processor.RegisterBusinessCmd(protocol.RouterMulticastForUniqId, r.ForUniqId)
}

// MulticastForUserIdParam 客户端发送的组播信息
type MulticastForUserIdParam struct {
	Message string
	UserIds []int `json:"userIds"`
}

// ForUserId 组播给某几个用户
func (multicast) ForUserId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(MulticastForUserIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse MulticastForUserIdParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	//查询目标用户的
	userIds := make([]string, 0)
	for _, userId := range payload.UserIds {
		user := userDb.Collect.GetUserById(userId)
		if user.IsOnline {
			userIds = append(userIds, strconv.Itoa(user.Id))
		}
	}
	//构建组播数据
	ret := &netsvrProtocol.Multicast{}
	//没有找到任何目标用户，通知发送方，目标用户不存在
	if len(userIds) == 0 {
		//目标用户不存在，返回信息给到发送者
		userIds = append(userIds, tf.UniqId)
		ret.Data = testUtils.NewResponse(protocol.RouterMulticastForUserId, map[string]interface{}{"code": 1, "message": "未找到目标用户"})
	} else {
		msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
		ret.Data = testUtils.NewResponse(protocol.RouterMulticastForUserId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	}
	ret.UniqIds = userIds
	processor.Send(ret, netsvrProtocol.Cmd_Multicast)
}

// MulticastForUniqIdParam 客户端发送的组播信息
type MulticastForUniqIdParam struct {
	Message string
	UnIqIds []string `json:"unIqIds"`
}

// ForUniqId 组播给某几个uniqId
func (multicast) ForUniqId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(MulticastForUniqIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse MulticastForUniqIdParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	//构建组播数据
	ret := &netsvrProtocol.Multicast{}
	ret.UniqIds = payload.UnIqIds
	msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
	ret.Data = testUtils.NewResponse(protocol.RouterMulticastForUniqId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	//发到网关
	processor.Send(ret, netsvrProtocol.Cmd_Multicast)
}
