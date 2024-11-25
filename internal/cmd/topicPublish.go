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
	"netsvr/internal/customer"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// TopicPublish 发布
func TopicPublish(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.TopicPublish.Get()
	defer objPool.TopicPublish.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicPublish failed")
		return
	}
	if len(payload.Data) == 0 {
		return
	}
	for _, t := range payload.Topics {
		if t == "" {
			continue
		}
		uniqIds := topic.Topic.GetUniqIds(t, objPool.UniqIdSlice)
		if uniqIds == nil {
			continue
		}
		uniqIdsAlias := *uniqIds //搞个别名，避免循环中解指针，提高性能
		//小于100个连接且topic数量最多两个，直接发送，这个条件的意思是：最多循环发送200个连接，超出限制都走协程并发发送
		if len(uniqIdsAlias) < 101 && len(payload.Topics) < 3 {
			for _, uniqId := range uniqIdsAlias {
				conn := manager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				customer.WriteMessage(conn, payload.Data)
			}
			objPool.UniqIdSlice.Put(uniqIds)
			//跳过，处理下一个topic
			continue
		}
		//大于100个连接，或者是topic数量大于2，开启多协程发送
		coroutineNum := len(uniqIdsAlias)/100 + 1
		connCh := make(chan *websocket.Conn, coroutineNum)
		for i := 0; i < coroutineNum; i++ {
			go func(data []byte, connCh chan *websocket.Conn) {
				defer func() {
					_ = recover()
				}()
				for conn := range connCh {
					customer.WriteMessage(conn, data)
				}
			}(payload.Data, connCh)
		}
		for _, uniqId := range uniqIdsAlias {
			conn := manager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			connCh <- conn
		}
		close(connCh)
		objPool.UniqIdSlice.Put(uniqIds)
	}
}
