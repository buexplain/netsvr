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
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
)

// topicCustomerIdList 获取网关中某几个主题的customerId
func topicCustomerIdList(param []byte, taskConn net.Conn) {
	payload := netsvrProtocol.TopicCustomerIdListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicCustomerIdListReq failed")
		return
	}
	ret := &netsvrProtocol.TopicCustomerIdListResp{Items: map[string]*netsvrProtocol.TopicCustomerIdListRespItem{}}
	topicConnList := topic.Topic.GetConnListByTopics(payload.Topics)
	for tpc, connList := range topicConnList {
		// 添加到返回结果中
		ret.Items[tpc] = &netsvrProtocol.TopicCustomerIdListRespItem{CustomerIds: customer.GetCustomerIds(connList)}
	}
	send(taskConn, ret, netsvrProtocol.Cmd_TopicCustomerIdList)
}
