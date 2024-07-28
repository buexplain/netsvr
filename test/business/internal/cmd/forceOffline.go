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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v3/netsvr"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type forceOffline struct{}

var ForceOffline = forceOffline{}

func (r forceOffline) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterForceOfflineByCustomerId, r.CustomerId)
	processor.RegisterBusinessCmd(protocol.RouterForceOffline, r.UniqId)
}

// ForceOfflineForCustomerIdParam 强制踢下线某个用户
type ForceOfflineForCustomerIdParam struct {
	CustomerIds []string `json:"customerIds"`
}

// CustomerId 强制关闭某个用户的连接
func (forceOffline) CustomerId(_ *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ForceOfflineForCustomerIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse ForceOfflineForCustomerIdParam failed")
		return
	}
	req := &netsvrProtocol.ForceOfflineByCustomerId{}
	req.CustomerIds = payload.CustomerIds
	req.Data = testUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"})
	processor.Send(req, netsvrProtocol.Cmd_ForceOfflineByCustomerId)
}

// ForceOfflineParam 强制踢下线某个连接
type ForceOfflineParam struct {
	UniqId string `json:"uniqId"`
}

// UniqId 强制关闭某个连接
func (forceOffline) UniqId(_ *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ForceOfflineParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse ForceOfflineParam failed")
		return
	}
	req := &netsvrProtocol.ForceOffline{}
	req.UniqIds = []string{payload.UniqId}
	req.Data = testUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"})
	processor.Send(req, netsvrProtocol.Cmd_ForceOffline)
}
