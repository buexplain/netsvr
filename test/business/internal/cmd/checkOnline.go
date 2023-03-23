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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"google.golang.org/protobuf/proto"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type checkOnline struct{}

var CheckOnline = checkOnline{}

func (r checkOnline) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterCheckOnlineForUniqId, r.RequestForUniqId)
	processor.RegisterWorkerCmd(protocol.RouterCheckOnlineForUniqId, r.ResponseForUniqId)
}

// CheckOnlineForUniqIdParam 检查某几个连接是否在线
type CheckOnlineForUniqIdParam struct {
	UniqIds []string `json:"uniqIds"`
}

// RequestForUniqId 向worker发起请求，检查某几个连接是否在线
func (checkOnline) RequestForUniqId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := CheckOnlineForUniqIdParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse CheckOnlineForUniqIdParam failed")
		return
	}
	req := &netsvrProtocol.CheckOnlineReq{}
	req.RouterCmd = int32(protocol.RouterCheckOnlineForUniqId)
	req.CtxData = testUtils.StrToReadOnlyBytes(tf.UniqId)
	req.UniqIds = payload.UniqIds
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_CheckOnline
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseForUniqId 处理worker的响应
func (checkOnline) ResponseForUniqId(param []byte, processor *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.CheckOnlineResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.CheckOnlineResp failed")
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := testUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = testUtils.NewResponse(protocol.RouterCheckOnlineForUniqId, map[string]interface{}{"code": 0, "message": "检查某几个连接是否在线成功", "data": payload.UniqIds})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
