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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"google.golang.org/protobuf/proto"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type metrics struct{}

var Metrics = metrics{}

func (r metrics) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterMetrics, r.Request)
}

// Request 获取网关统计的服务状态
func (metrics) Request(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	resp := &netsvrProtocol.MetricsResp{}
	testUtils.RequestNetSvr(nil, netsvrProtocol.Cmd_Metrics, resp)
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	ret.Data = testUtils.NewResponse(protocol.RouterMetrics, map[string]interface{}{"code": 0, "message": "获取网关状态的统计信息成功", "data": resp.Items})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
