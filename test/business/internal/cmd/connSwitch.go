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
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	"netsvr/test/pkg/utils"
)

type connSwitch struct{}

var ConnSwitch = connSwitch{}

// ConnOpen 客户端打开连接
func (connSwitch) ConnOpen(payload *netsvrProtocol.ConnOpen) {
	data := utils.NewResponse(protocol.RouterRespConnOpen, map[string]interface{}{
		"code":    0,
		"message": "连接网关成功",
		"data": map[string]interface{}{
			"uniqId":        payload.UniqId,
			"rawQuery":      payload.RawQuery,
			"subProtocol":   payload.SubProtocol,
			"xForwardedFor": payload.XForwardedFor,
			"xRealIp":       payload.XRealIp,
			"remoteAddr":    payload.RemoteAddr,
		},
	})
	//发送到网关
	netBus.NetBus.SingleCast(payload.UniqId, data)
}

// ConnClose 客户端关闭连接
func (connSwitch) ConnClose(payload *netsvrProtocol.ConnClose) {
	//解析网关中存储的用户信息
	user := userDb.ParseNetSvrInfo(payload.Session)
	if user != nil {
		//更新数据库，标记用户已经下线
		userDb.Collect.SetOnlineInfo(user.Id, "")
	}
}
