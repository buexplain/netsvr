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
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type singleCast struct{}

var SingleCast = singleCast{}

func init() {
	businessCmdCallback[protocol.RouterSingleCastByCustomerId] = SingleCast.CustomerId
	businessCmdCallback[protocol.RouterSingleCast] = SingleCast.UniqId
}

// SingleCastByCustomerIdParam 客户端发送的单播信息
type SingleCastByCustomerIdParam struct {
	Message    string `json:"message"`
	CustomerId string `json:"customerId"`
}

// CustomerId 单播个某个用户
func (singleCast) CustomerId(tf *netsvrProtocol.Transfer, param string) {
	//解析客户端发来的数据
	payload := SingleCastByCustomerIdParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse SingleCastByCustomerIdParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
	netBus.NetBus.SingleCastByCustomerId(payload.CustomerId, testUtils.NewResponse(protocol.RouterSingleCastByCustomerId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg}))
}

// SingleCastParam 客户端发送的单播信息
type SingleCastParam struct {
	Message string
	UniqId  string `json:"uniqId"`
}

// UniqId 单播给某个uniqId
func (singleCast) UniqId(tf *netsvrProtocol.Transfer, param string) {
	payload := SingleCastParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse SingleCastParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	//构建单播数据
	msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
	//发到网关
	netBus.NetBus.SingleCast(payload.UniqId, testUtils.NewResponse(protocol.RouterSingleCast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg}))
}
