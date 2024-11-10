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
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// TopicUniqIdCount 获取网关中的主题包含的连接数
func TopicUniqIdCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := netsvrProtocol.TopicUniqIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicUniqIdCountReq failed")
		return
	}
	ret := &netsvrProtocol.TopicUniqIdCountResp{}
	ret.Items = map[string]int32{}
	if payload.CountAll == true {
		topic.Topic.CountAll(ret.Items)
	} else {
		topic.Topic.Count(payload.Topics, ret.Items)
	}
	processor.Send(ret, netsvrProtocol.Cmd_TopicUniqIdCount)
}
