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
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// TopicPublishBulk 批量发布
func TopicPublishBulk(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.TopicPublishBulk.Get()
	if err := proto.Unmarshal(param, payload); err != nil {
		objPool.TopicPublishBulk.Put(payload)
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicPublishBulk failed")
		return
	}
	//如果结构不对称，则会引起下面代码的切片索引越界，所以丢弃不做处理
	if len(payload.Data) != len(payload.Topics) {
		objPool.TopicPublishBulk.Put(payload)
		return
	}
	for index, t := range payload.Topics {
		if t == "" {
			continue
		}
		dataLen := int64(len(payload.Data[index]))
		if dataLen == 0 {
			continue
		}
		uniqIds := topic.Topic.GetUniqIds(t, objPool.UniqIdSlice)
		if uniqIds == nil {
			continue
		}
		for _, uniqId := range *uniqIds {
			conn := manager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, payload.Data[index]); err == nil {
				metrics.Registry[metrics.ItemCustomerWriteNumber].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(dataLen)
			} else {
				_ = conn.Close()
			}
		}
		objPool.UniqIdSlice.Put(uniqIds)
	}
	objPool.TopicPublishBulk.Put(payload)
}
