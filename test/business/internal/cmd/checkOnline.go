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

type checkOnline struct{}

var CheckOnline = checkOnline{}

func init() {
	businessCmdCallback[protocol.RouterCheckOnline] = CheckOnline.UniqId
}

// CheckOnlineParam 检查某几个连接是否在线
type CheckOnlineParam struct {
	UniqIds []string `json:"uniqIds"`
}

// UniqId 向worker发起请求，检查某几个连接是否在线
func (checkOnline) UniqId(tf *netsvrProtocol.Transfer, param string) {
	payload := CheckOnlineParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse CheckOnlineParam failed")
		return
	}
	resp := netBus.NetBus.CheckOnline(payload.UniqIds)
	//将结果单播给客户端
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterCheckOnline, map[string]interface{}{"code": 0, "message": "检查某几个连接是否在线成功", "data": resp.Data}))
}
