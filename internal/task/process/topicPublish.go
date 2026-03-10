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
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/customer"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
)

// topicPublish 发布
func topicPublish(param []byte) {
	payload := objPool.TopicPublish.Get()
	defer objPool.TopicPublish.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.topicPublish failed")
		return
	}
	if len(payload.Data) == 0 {
		return
	}
	msg := customer.NewMessage(configs.Config.Customer.SendMessageType, payload.Data)
	for _, t := range payload.Topics {
		if t == "" {
			continue
		}
		uniqIds := topic.Topic.GetUniqIds(t, objPool.UniqIdSlice)
		if uniqIds == nil {
			continue
		}
		uniqIdsAlias := *uniqIds //搞个别名，避免循环中解指针，提高性能
		for _, uniqId := range uniqIdsAlias {
			conn := manager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			msg.WriteTo(conn)
		}
		objPool.UniqIdSlice.Put(uniqIds)
	}
}
