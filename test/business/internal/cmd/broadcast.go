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
	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
	"encoding/json"
	"fmt"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type broadcast struct{}

var Broadcast = broadcast{}

func init() {
	businessCmdCallback[protocol.RouterBroadcast] = Broadcast.Request
	businessCmdCallback[protocol.RouterBroadcastBulk] = Broadcast.Bulk
}

// BroadcastParam 客户端发送的广播信息
type BroadcastParam struct {
	Message string
}

// BroadcastBulkParam 客户端发送的批量广播信息
type BroadcastBulkParam struct {
	Message []string
}

// Request 向worker发起广播请求
func (broadcast) Request(tf *netsvrProtocol.Transfer, param string) {
	target := new(BroadcastParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), target); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse BroadcastParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	msg := map[string]interface{}{"fromUser": fromUser, "message": target.Message}
	netBus.NetBus.Broadcast(testUtils.NewResponse(protocol.RouterBroadcast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg}))
}

// Bulk 向worker发起批量广播请求：按顺序把每一条数据广播给全部连接
func (broadcast) Bulk(tf *netsvrProtocol.Transfer, param string) {
	target := new(BroadcastBulkParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), target); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse BroadcastBulkParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	data := make([][]byte, 0, len(target.Message))
	for _, item := range target.Message {
		//空消息按空数据下发，网关会跳过零长度的数据项
		if item == "" {
			data = append(data, nil)
			continue
		}
		msg := map[string]interface{}{"fromUser": fromUser, "message": item}
		data = append(data, testUtils.NewResponse(protocol.RouterBroadcastBulk, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg}))
	}
	netBus.NetBus.BroadcastBulk(data)
}
