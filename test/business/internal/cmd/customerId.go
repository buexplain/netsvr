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
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
	"netsvr/test/pkg/utils/netSvrPool"
)

type customerId struct{}

var CustomerId = customerId{}

func (r customerId) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterCustomerIdList, r.RequestList)
	processor.RegisterBusinessCmd(protocol.RouterCustomerIdCount, r.RequestCount)
}

// RequestList 获取网关所有的customerId
func (customerId) RequestList(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	resp := &netsvrProtocol.CustomerIdListResp{}
	netSvrPool.Request(nil, netsvrProtocol.Cmd_CustomerIdList, resp)
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{
		"customerIds": resp.CustomerIds,
	}
	ret.Data = testUtils.NewResponse(protocol.RouterCustomerIdList, map[string]interface{}{"code": 0, "message": "获取网关所有的customerId成功", "data": msg})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}

// RequestCount 获取网关中customerId的数量
func (customerId) RequestCount(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	resp := &netsvrProtocol.CustomerIdCountResp{}
	netSvrPool.Request(nil, netsvrProtocol.Cmd_CustomerIdCount, resp)
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{
		"count": resp.Count,
	}
	ret.Data = testUtils.NewResponse(protocol.RouterCustomerIdCount, map[string]interface{}{"code": 0, "message": "获取网关中customerId的数量成功", "data": msg})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}
