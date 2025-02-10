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
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/binder"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// TopicCustomerIdCount 获取网关中目标topic的customerId数量
func TopicCustomerIdCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := netsvrProtocol.TopicCustomerIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicCustomerIdCountReq failed")
		return
	}
	ret := &netsvrProtocol.TopicCustomerIdCountResp{}
	ret.Items = map[string]int32{}
	//统计全部主题
	if payload.CountAll == true {
		payload.Topics = topic.Topic.Get()
	}
	// 获取topic的uniqId
	topicUniqId := topic.Topic.GetUniqIdsByTopics(payload.Topics)
	for tpc, uniqIds := range topicUniqId {
		// 获取topic的uniqId的customerId数量
		ret.Items[tpc] = int32(binder.Binder.CountCustomerIdsByUniqIds(uniqIds))
	}
	processor.Send(ret, netsvrProtocol.Cmd_TopicCustomerIdCount)
}
