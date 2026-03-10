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
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
)

// topicUniqIdCount 获取网关中的主题包含的连接数
func topicUniqIdCount(param []byte, taskConn net.Conn) {
	payload := netsvrProtocol.TopicUniqIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicUniqIdCountReq failed")
		return
	}
	ret := &netsvrProtocol.TopicUniqIdCountResp{}
	if payload.CountAll == true {
		ret.Items = topic.Topic.CountAll()
	} else {
		ret.Items = topic.Topic.Count(payload.Topics)
	}
	send(taskConn, ret, netsvrProtocol.Cmd_TopicUniqIdCount)
}
