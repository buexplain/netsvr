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
	"netsvr/internal/wsServer"
)

// topicCustomerIdCount 获取网关中目标topic的customerId数量
func topicCustomerIdCount(param []byte, taskConn net.Conn) {
	payload := netsvrProtocol.TopicCustomerIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicCustomerIdCountReq failed")
		return
	}
	ret := &netsvrProtocol.TopicCustomerIdCountResp{}
	ret.Items = map[string]int32{}
	var topicConnList map[string][]*wsServer.Conn
	if payload.CountAll == true {
		topicConnList = topic.Topic.GetConnList()
	} else {
		topicConnList = topic.Topic.GetConnListByTopics(payload.Topics)
	}
	for tpc, connList := range topicConnList {
		ret.Items[tpc] = int32(customer.CountCustomerIds(connList))
	}
	send(taskConn, ret, netsvrProtocol.Cmd_TopicCustomerIdCount)
}
