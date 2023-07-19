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

type limit struct{}

var Limit = limit{}

func (r limit) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterLimit, r.Request)
}

// LimitUpdateParam 更新网关中限流配置
type LimitUpdateParam struct {
	Concurrency int32  `json:"concurrency"`
	Name        string `json:"name"`
}

// Request 更新限流配置、获取网关中的限流配置的真实情况
func (limit) Request(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(LimitUpdateParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse LimitUpdateParam failed")
		return
	}
	req := &netsvrProtocol.LimitReq{}
	req.Items = []*netsvrProtocol.LimitReqItem{
		{
			Name:        payload.Name,
			Concurrency: payload.Concurrency,
		},
	}
	resp := &netsvrProtocol.LimitResp{}
	testUtils.RequestNetSvr(req, netsvrProtocol.Cmd_Limit, resp)
	//将结果单播给客户端
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	ret.Data = testUtils.NewResponse(protocol.RouterLimit, map[string]interface{}{"code": 0, "message": "获取网关中的限流配置的真实情况成功", "data": resp.Items})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
