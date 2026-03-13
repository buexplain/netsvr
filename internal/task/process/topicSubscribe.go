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
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
)

// topicSubscribe 订阅
func topicSubscribe(param []byte) {
	payload := objPool.TopicSubscribe.Get()
	defer objPool.TopicSubscribe.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.topicSubscribe failed")
		return
	}
	if payload.UniqId == "" || len(payload.Topics) == 0 {
		return
	}
	conn := customerManager.Manager.Get(payload.UniqId)
	session, ok := customer.GetSession(conn)
	if !ok {
		return
	}
	session.Lock()
	defer session.UnLock()
	if session.GetUniqId() == "" {
		return
	}
	session.SubscribeTopics(payload.Topics)
	topic.Topic.SetBySlice(payload.Topics, payload.UniqId)
	if len(payload.Data) > 0 {
		customer.WriteMessage(conn, configs.Config.Customer.SendMessageType, payload.Data)
	}
}
