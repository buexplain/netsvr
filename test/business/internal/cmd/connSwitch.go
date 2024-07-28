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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v3/netsvr"
	"google.golang.org/protobuf/proto"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	"netsvr/test/pkg/utils"
)

type connSwitch struct{}

var ConnSwitch = connSwitch{}

func (r connSwitch) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterWorkerCmd(netsvrProtocol.Cmd_ConnOpen, r.ConnOpen)
	processor.RegisterWorkerCmd(netsvrProtocol.Cmd_ConnClose, r.ConnClose)
}

// ConnOpen 客户端打开连接
func (connSwitch) ConnOpen(param []byte, processor *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.ConnOpen{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.ConnOpen failed")
		return
	}
	//构造单播数据
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = payload.UniqId
	ret.Data = utils.NewResponse(protocol.RouterRespConnOpen, map[string]interface{}{
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
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}

// ConnClose 客户端关闭连接
func (connSwitch) ConnClose(param []byte, _ *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.ConnClose{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.ConnClose failed")
		return
	}
	//解析网关中存储的用户信息
	user := userDb.ParseNetSvrInfo(payload.Session)
	if user != nil {
		//更新数据库，标记用户已经下线
		userDb.Collect.SetOnlineInfo(user.Id, "")
	}
}
