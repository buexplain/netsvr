/**
* Copyright 2022 buexplain@qq.com
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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/protocol"
	"google.golang.org/protobuf/proto"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
	"strconv"
)

type forceOffline struct{}

var ForceOffline = forceOffline{}

func (r forceOffline) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterForceOfflineForUserId, r.ForUserId)
	processor.RegisterBusinessCmd(protocol.RouterForceOfflineForUniqId, r.ForUniqId)
}

// ForceOfflineForUserIdParam 强制踢下线某个用户
type ForceOfflineForUserIdParam struct {
	UserId int `json:"userId"`
}

// ForUserId 强制关闭某个用户的连接
func (forceOffline) ForUserId(_ *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ForceOfflineForUserIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse ForceOfflineForUserIdParam failed")
		return
	}
	user := userDb.Collect.GetUserById(payload.UserId)
	if user == nil || user.IsOnline == false {
		return
	}
	ret := &netsvrProtocol.ForceOffline{}
	ret.UniqIds = []string{strconv.Itoa(user.Id)}
	ret.Data = testUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_ForceOffline
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ForceOfflineForUniqIdParam 强制踢下线某个用户
type ForceOfflineForUniqIdParam struct {
	UniqId string `json:"uniqId"`
}

// ForUniqId 强制关闭某个用户的连接
func (forceOffline) ForUniqId(_ *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ForceOfflineForUniqIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse ForceOfflineForUniqIdParam failed")
		return
	}
	ret := &netsvrProtocol.ForceOffline{}
	ret.UniqIds = []string{payload.UniqId}
	ret.Data = testUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_ForceOffline
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
