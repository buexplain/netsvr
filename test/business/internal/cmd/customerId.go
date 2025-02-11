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
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type customerId struct{}

var CustomerId = customerId{}

func init() {
	businessCmdCallback[protocol.RouterCustomerIdList] = CustomerId.RequestList
	businessCmdCallback[protocol.RouterCustomerIdCount] = CustomerId.RequestCount
}

// RequestList 获取网关所有的customerId
func (customerId) RequestList(tf *netsvrProtocol.Transfer, _ string) {
	resp := netBus.NetBus.CustomerIdList()
	//将结果单播给客户端
	msg := map[string]interface{}{
		"customerIds": resp.Data,
	}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterCustomerIdList, map[string]interface{}{"code": 0, "message": "获取网关所有的customerId成功", "data": msg}))
}

// RequestCount 获取网关中customerId的数量
func (customerId) RequestCount(tf *netsvrProtocol.Transfer, _ string) {
	resp := netBus.NetBus.CustomerIdCount()
	//将结果单播给客户端
	msg := map[string]interface{}{
		"count": resp.Data,
	}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterCustomerIdCount, map[string]interface{}{"code": 0, "message": "获取网关中customerId的数量成功", "data": msg}))
}
