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

type uniqId struct{}

var UniqId = uniqId{}

func init() {
	businessCmdCallback[protocol.RouterUniqIdList] = UniqId.RequestList
	businessCmdCallback[protocol.RouterUniqIdCount] = UniqId.RequestCount
}

// RequestList 获取网关所有的uniqId
func (uniqId) RequestList(tf *netsvrProtocol.Transfer, _ string) {
	resp := netBus.NetBus.UniqIdList()
	//将结果单播给客户端
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterUniqIdList, map[string]interface{}{"code": 0, "message": "获取网关所有的uniqId成功", "data": resp.Data}))
}

// RequestCount 获取网关中uniqId的数量
func (uniqId) RequestCount(tf *netsvrProtocol.Transfer, _ string) {
	resp := netBus.NetBus.UniqIdCount()
	//将结果单播给客户端
	msg := map[string]interface{}{
		"count": resp.Count(),
	}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterUniqIdCount, map[string]interface{}{"code": 0, "message": "获取网关中uniqId的数量成功", "data": msg}))
}
