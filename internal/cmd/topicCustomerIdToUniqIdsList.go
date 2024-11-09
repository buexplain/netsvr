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
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/binder"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// TopicCustomerIdToUniqIdsList 获取网关中目标topic的customerId以及对应的uniqId列表
func TopicCustomerIdToUniqIdsList(param []byte, processor *workerManager.ConnProcessor) {
	payload := netsvrProtocol.TopicCustomerIdToUniqIdsListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicCustomerIdToUniqIdsListReq failed")
		return
	}
	ret := &netsvrProtocol.TopicCustomerIdToUniqIdsListResp{
		Items: map[string]*netsvrProtocol.TopicCustomerIdToUniqIdsListRespItem{},
	}
	// 获取topic的uniqId
	topicUniqId := topic.Topic.GetUniqIdsByTopics(payload.Topics)
	for tpc, uniqIds := range topicUniqId {
		// 获取topic的uniqId的customerId
		customerIdToUniqIdsList := binder.Binder.GetCustomerIdToUniqIdsList(uniqIds)
		items := &netsvrProtocol.TopicCustomerIdToUniqIdsListRespItem{
			Items: make(map[string]*netsvrProtocol.CustomerIdToUniqIdsRespItem, len(customerIdToUniqIdsList)),
		}
		//转换为返回结果
		for customerId, uniqIds := range customerIdToUniqIdsList {
			items.Items[customerId] = &netsvrProtocol.CustomerIdToUniqIdsRespItem{
				UniqIds: uniqIds,
			}
		}
		// 添加到返回结果中
		ret.Items[tpc] = items
	}
	processor.Send(ret, netsvrProtocol.Cmd_TopicCustomerIdToUniqIdsList)
}
