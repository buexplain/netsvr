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

package process

import (
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"google.golang.org/protobuf/proto"
	"net"
	"netsvr/internal/customer"
	"netsvr/internal/customer/binder"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
)

// topicCustomerIdToUniqIdsList 获取网关中目标topic的customerId以及对应的uniqId列表
func topicCustomerIdToUniqIdsList(param []byte, taskConn net.Conn) {
	payload := netsvrProtocol.TopicCustomerIdToUniqIdsListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicCustomerIdToUniqIdsListReq failed")
		return
	}
	ret := &netsvrProtocol.TopicCustomerIdToUniqIdsListResp{
		Items: map[string]*netsvrProtocol.TopicCustomerIdToUniqIdsListRespItem{},
	}
	topicConnList := topic.Topic.GetConnListByTopics(payload.Topics)
	for tpc, connList := range topicConnList {
		customerIdConnList := binder.Binder.GetConnListByCustomerIds(customer.GetCustomerIds(connList))
		items := &netsvrProtocol.TopicCustomerIdToUniqIdsListRespItem{
			Items: make(map[string]*netsvrProtocol.CustomerIdToUniqIdsRespItem, len(customerIdConnList)),
		}
		//转换为返回结果
		for customerId, connList2 := range customerIdConnList {
			items.Items[customerId] = &netsvrProtocol.CustomerIdToUniqIdsRespItem{
				UniqIds: customer.GetUniqIds(connList2),
			}
		}
		// 添加到返回结果中
		ret.Items[tpc] = items
	}
	send(taskConn, ret, netsvrProtocol.Cmd_TopicCustomerIdToUniqIdsList)
}
