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
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// TopicUnsubscribe 取消订阅
func TopicUnsubscribe(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.TopicUnsubscribe.Get()
	defer objPool.TopicUnsubscribe.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicUnsubscribe failed")
		return
	}
	if payload.UniqId == "" || len(payload.Topics) == 0 {
		return
	}
	conn := customerManager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	session, ok := conn.SessionWithLock().(*info.Info)
	if !ok {
		return
	}
	session.Lock()
	defer session.UnLock()
	if session.GetUniqId() == "" {
		return
	}
	session.UnsubscribeTopics(payload.Topics)
	topic.Topic.DelBySlice(payload.Topics, payload.UniqId)
	if len(payload.Data) > 0 {
		customer.WriteMessage(conn, payload.Data)
	}
}
