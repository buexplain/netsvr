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

type limit struct{}

var Limit = limit{}

func init() {
	businessCmdCallback[protocol.RouterLimit] = Limit.Request
}

// LimitUpdateParam 更新网关中限流配置
type LimitUpdateParam struct {
	// 网关允许每秒转发多少个连接打开事件到business进程
	OnOpen int32 `json:"onOpen"`
	// 网关允许每秒转发多少个消息到business进程
	OnMessage int32 `json:"onMessage"`
}

// Request 更新限流配置、获取网关中的限流配置的真实情况
func (limit) Request(tf *netsvrProtocol.Transfer, param string) {
	payload := new(LimitUpdateParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse LimitUpdateParam failed")
		return
	}
	req := netsvrProtocol.LimitReq{}
	req.OnOpen = payload.OnOpen
	req.OnMessage = payload.OnMessage
	resp := netBus.NetBus.Limit(&req, "")
	//将结果单播给客户端
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterLimit, map[string]interface{}{
		"code":    0,
		"message": "获取网关中的限流配置的真实情况成功",
		"data":    resp.Data,
	}))
}
