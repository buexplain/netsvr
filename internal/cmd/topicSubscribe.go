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
	"netsvr/internal/metrics"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// TopicSubscribe 订阅
func TopicSubscribe(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.TopicSubscribe.Get()
	if err := proto.Unmarshal(param, payload); err != nil {
		objPool.TopicSubscribe.Put(payload)
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicSubscribe failed")
		return
	}
	if payload.UniqId == "" || len(payload.Topics) == 0 {
		objPool.TopicSubscribe.Put(payload)
		return
	}
	conn := customerManager.Manager.Get(payload.UniqId)
	if conn == nil {
		objPool.TopicSubscribe.Put(payload)
		return
	}
	session, ok := conn.Session().(*info.Info)
	if !ok {
		objPool.TopicSubscribe.Put(payload)
		return
	}
	session.MuxLock()
	payload.UniqId = session.SubscribeTopics(payload.Topics)
	//这里根据session里面的uniqId去构建订阅关系，因为有可能当SubscribeTopics得到锁的时候，session里面的uniqId与当前的payload.UniqId不一致了
	topic.Topic.SetBySlice(payload.Topics, payload.UniqId)
	session.MuxUnLock()
	if len(payload.Data) > 0 {
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err == nil {
			metrics.Registry[metrics.ItemCustomerWriteNumber].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(payload.Data)))
		} else {
			_ = conn.Close()
		}
	}
	objPool.TopicSubscribe.Put(payload)
}
