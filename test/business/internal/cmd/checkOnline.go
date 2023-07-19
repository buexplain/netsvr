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
}

// CheckOnlineForUniqIdParam 检查某几个连接是否在线
type CheckOnlineForUniqIdParam struct {
	UniqIds []string `json:"uniqIds"`
}

// RequestForUniqId 向worker发起请求，检查某几个连接是否在线
func (checkOnline) RequestForUniqId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := CheckOnlineForUniqIdParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse CheckOnlineForUniqIdParam failed")
		return
	}
	req := &netsvrProtocol.CheckOnlineReq{}
	req.UniqIds = payload.UniqIds
	resp := &netsvrProtocol.CheckOnlineResp{}
	testUtils.RequestNetSvr(req, netsvrProtocol.Cmd_CheckOnline, resp)
	//将结果单播给客户端
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	ret.Data = testUtils.NewResponse(protocol.RouterCheckOnlineForUniqId, map[string]interface{}{"code": 0, "message": "检查某几个连接是否在线成功", "data": resp.UniqIds})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
