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
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type forceOfflineGuest struct{}

var ForceOfflineGuest = forceOfflineGuest{}

func (r forceOfflineGuest) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterForceOfflineGuest, r.UniqId)
}

// ForceOfflineGuestParam 将某个没有session值的连接强制关闭
type ForceOfflineGuestParam struct {
	UniqId string `json:"uniqId"`
	Delay  int32  `json:"delay"`
}

// UniqId 将某个没有session值的连接强制关闭
func (forceOfflineGuest) UniqId(_ *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ForceOfflineGuestParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse ForceOfflineGuestParam failed")
		return
	}
	ret := &netsvrProtocol.ForceOfflineGuest{}
	ret.UniqIds = []string{payload.UniqId}
	ret.Delay = payload.Delay
	ret.Data = testUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "游客您好，您已被迫下线！"})
	processor.Send(ret, netsvrProtocol.Cmd_ForceOfflineGuest)
}
