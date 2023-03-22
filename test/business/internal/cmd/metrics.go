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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/protocol"
	"google.golang.org/protobuf/proto"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type metrics struct{}

var Metrics = metrics{}

func (r metrics) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterMetrics, r.Request)
	processor.RegisterWorkerCmd(protocol.RouterMetrics, r.Response)
}

// Request 获取网关统计的服务状态
func (metrics) Request(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &netsvrProtocol.MetricsReq{}
	req.RouterCmd = int32(protocol.RouterMetrics)
	req.CtxData = testUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_Metrics
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// Response 处理worker发送过来的网关统计的服务状态
func (metrics) Response(param []byte, processor *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.MetricsResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.MetricsResp failed")
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := testUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = testUtils.NewResponse(protocol.RouterMetrics, map[string]interface{}{"code": 0, "message": "获取网关状态的统计信息成功", "data": payload.Items})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
