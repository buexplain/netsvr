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
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type multicast struct{}

var Multicast = multicast{}

func (r multicast) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterMulticast, r.UniqId)
	processor.RegisterBusinessCmd(protocol.RouterMulticastByCustomerId, r.CustomerId)
}

// MulticastForUserIdParam 客户端发送的组播信息
type MulticastForUserIdParam struct {
	Message string
	UserIds []uint32 `json:"userIds"`
}

// MulticastParam 客户端发送的组播信息
type MulticastParam struct {
	Message string
	UnIqIds []string `json:"unIqIds"`
}

// UniqId 组播给某几个uniqId
func (multicast) UniqId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(MulticastParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse MulticastParam failed")
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
	ret.Data = testUtils.NewResponse(protocol.RouterMulticast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	//发到网关
	processor.Send(ret, netsvrProtocol.Cmd_Multicast)
}

// CustomerIdParam 客户端发送的组播信息
type CustomerIdParam struct {
	Message     string
	CustomerIds []string `json:"customerIds"`
}

// CustomerId 组播给某几个customerId
func (multicast) CustomerId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(CustomerIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse CustomerIdParam failed")
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
	ret := &netsvrProtocol.MulticastByCustomerId{}
	ret.CustomerIds = payload.CustomerIds
	msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
	ret.Data = testUtils.NewResponse(protocol.RouterMulticastByCustomerId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	//发到网关
	processor.Send(ret, netsvrProtocol.Cmd_MulticastByCustomerId)
}
