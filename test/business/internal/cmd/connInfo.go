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

type connInfo struct{}

var ConnInfo = connInfo{}

func init() {
	businessCmdCallback[protocol.RouterConnInfo] = ConnInfo.RequestConnInfo
	businessCmdCallback[protocol.RouterConnInfoByCustomerId] = ConnInfo.RequestConnInfoByCustomerId
}

// RequestConnInfo 获取我的连接信息
func (connInfo) RequestConnInfo(tf *netsvrProtocol.Transfer, _ string) {
	resp := netBus.NetBus.ConnInfo([]string{tf.UniqId}, true, true, true)
	msg := map[string]interface{}{"connInfo": resp.Data}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterConnInfo, map[string]interface{}{"code": 0, "message": "获取我的连接信息成功", "data": msg}))
}

// ConnInfoByCustomerIdParam 强制踢下线某个用户
type ConnInfoByCustomerIdParam struct {
	CustomerIds []string `json:"customerIds"`
}

// RequestConnInfoByCustomerId 获取customerId的连接信息
func (connInfo) RequestConnInfoByCustomerId(tf *netsvrProtocol.Transfer, param string) {
	payload := new(ConnInfoByCustomerIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse ConnInfoByCustomerIdParam failed")
		return
	}
	resp := netBus.NetBus.ConnInfoByCustomerId(payload.CustomerIds, true, true, true)
	msg := map[string]interface{}{"connInfo": resp.Data}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterConnInfoByCustomerId, map[string]interface{}{"code": 0, "message": "获取customerId的连接信息成功", "data": msg}))
}
