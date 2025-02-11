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
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type forceOffline struct{}

var ForceOffline = forceOffline{}

func init() {
	businessCmdCallback[protocol.RouterForceOfflineByCustomerId] = ForceOffline.CustomerId
	businessCmdCallback[protocol.RouterForceOffline] = ForceOffline.UniqId
}

// ForceOfflineForCustomerIdParam 强制踢下线某个用户
type ForceOfflineForCustomerIdParam struct {
	CustomerIds []string `json:"customerIds"`
}

// CustomerId 强制关闭某个用户的连接
func (forceOffline) CustomerId(_ *netsvrProtocol.Transfer, param string) {
	payload := new(ForceOfflineForCustomerIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse ForceOfflineForCustomerIdParam failed")
		return
	}
	netBus.NetBus.ForceOfflineByCustomerId(payload.CustomerIds, testUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"}))
}

// ForceOfflineParam 强制踢下线某个连接
type ForceOfflineParam struct {
	UniqId string `json:"uniqId"`
}

// UniqId 强制关闭某个连接
func (forceOffline) UniqId(_ *netsvrProtocol.Transfer, param string) {
	payload := new(ForceOfflineParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse ForceOfflineParam failed")
		return
	}
	netBus.NetBus.ForceOffline([]string{payload.UniqId}, testUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"}))
}
