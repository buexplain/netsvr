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
	"netsvr/configs"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
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
	dataLen := int64(len(payload.Data))
	if dataLen == 0 {
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
		for _, uniqId := range uniqIdsAlias {
			conn := manager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			if err := conn.WriteMessage(configs.Config.Customer.SendMessageType, payload.Data); err == nil {
				metrics.Registry[metrics.ItemCustomerWriteNumber].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(dataLen)
			} else {
				_ = conn.Close()
			}
		}
		objPool.UniqIdSlice.Put(uniqIds)
	}
}
