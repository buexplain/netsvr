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

// topicUniqIdList 获取网关中某个主题包含的uniqId
func topicUniqIdList(param []byte, taskConn net.Conn) {
	payload := netsvrProtocol.TopicUniqIdListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicUniqIdListReq failed")
		return
	}
	ret := &netsvrProtocol.TopicUniqIdListResp{Items: map[string]*netsvrProtocol.TopicUniqIdListRespItem{}}
	topicConnList := topic.Topic.GetConnListByTopics(payload.Topics)
	for tpc, connList := range topicConnList {
		item := &netsvrProtocol.TopicUniqIdListRespItem{}
		item.UniqIds = customer.GetUniqIds(connList)
		ret.Items[tpc] = item
	}
	send(taskConn, ret, netsvrProtocol.Cmd_TopicUniqIdList)
}
