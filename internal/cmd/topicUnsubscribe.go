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
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
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
	if err := proto.Unmarshal(param, payload); err != nil {
		objPool.TopicUnsubscribe.Put(payload)
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicUnsubscribe failed")
		return
	}
	if payload.UniqId == "" || len(payload.Topics) == 0 {
		objPool.TopicUnsubscribe.Put(payload)
		return
	}
	conn := customerManager.Manager.Get(payload.UniqId)
	if conn == nil {
		objPool.TopicUnsubscribe.Put(payload)
		return
	}
	session, ok := conn.Session().(*info.Info)
	if !ok {
		objPool.TopicUnsubscribe.Put(payload)
		return
	}
	session.MuxLock()
	currentUniqId := session.UnsubscribeTopics(payload.Topics)
	//这里根据session里面的uniqId去删除订阅关系，因为有可能当UnsubscribeTopics得到锁的时候，session里面的uniqId与当前的payload.UniqId不一致了
	topic.Topic.DelBySlice(payload.Topics, currentUniqId, payload.UniqId)
	session.MuxUnLock()
	if len(payload.Data) > 0 {
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			_ = conn.Close()
		}
	}
	objPool.TopicUnsubscribe.Put(payload)
}
