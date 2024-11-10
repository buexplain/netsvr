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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/binder"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// TopicCustomerIdList 获取网关中某几个主题的customerId
func TopicCustomerIdList(param []byte, processor *workerManager.ConnProcessor) {
	payload := netsvrProtocol.TopicCustomerIdListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicCustomerIdListReq failed")
		return
	}
	ret := &netsvrProtocol.TopicCustomerIdListResp{Items: map[string]*netsvrProtocol.TopicCustomerIdListRespItem{}}
	// 获取topic的uniqId
	topicUniqId := topic.Topic.GetUniqIdsByTopics(payload.Topics)
	for tpc, uniqIds := range topicUniqId {
		// 获取topic的uniqId的customerId
		customerIds := binder.Binder.GetCustomerIdsByUniqIds(uniqIds)
		// 添加到返回结果中
		ret.Items[tpc] = &netsvrProtocol.TopicCustomerIdListRespItem{CustomerIds: customerIds}
	}
	processor.Send(ret, netsvrProtocol.Cmd_TopicCustomerIdList)
}
