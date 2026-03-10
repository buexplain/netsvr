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

// topicPublishBulk 批量发布
func topicPublishBulk(param []byte) {
	payload := objPool.TopicPublishBulk.Get()
	defer objPool.TopicPublishBulk.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.topicPublishBulk failed")
		return
	}
	//当业务进程传递的topics的topic数量只有一个，data的datum数量是一个以上时，网关必须将所有的datum都发送给这个topic
	if len(payload.Topics) == 1 && len(payload.Data) > 1 {
		//先根据主题，获得主题下的所有uniqId
		uniqIds := topic.Topic.GetUniqIds(payload.Topics[0], objPool.UniqIdSlice)
		if uniqIds == nil {
			return
		}
		defer objPool.UniqIdSlice.Put(uniqIds)
		//再迭代所有uniqId
		uniqIdsAlias := *uniqIds //搞个别名，避免循环中解指针，提高性能
		for _, data := range payload.Data {
			msg := customer.NewMessage(configs.Config.Customer.SendMessageType, data)
			for _, uniqId := range uniqIdsAlias {
				conn := manager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				msg.WriteTo(conn)
			}
		}
		return
	}
	//当业务进程传递的topics的topic数量与data的datum数量一致时，网关必须将同一下标的datum，发送给同一下标的topic
	if len(payload.Topics) > 0 && len(payload.Topics) == len(payload.Data) {
		var index int
		var currentTopic string
		var uniqId string
		//迭代所有的主题
		for index, currentTopic = range payload.Topics {
			//判断当前迭代的主题对应的数据是否有效
			if len(payload.Data[index]) == 0 {
				continue
			}
			//获得当前迭代的主题下的所有uniqId
			uniqIds := topic.Topic.GetUniqIds(currentTopic, objPool.UniqIdSlice)
			if uniqIds == nil {
				continue
			}
			//迭代所有uniqId
			uniqIdsAlias := *uniqIds //搞个别名，避免循环中解指针，提高性能
			msg := customer.NewMessage(configs.Config.Customer.SendMessageType, payload.Data[index])
			for _, uniqId = range uniqIdsAlias {
				//根据uniqId获得对应的连接
				conn := manager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				//将当前迭代的主题对应的数据写入到该连接
				msg.WriteTo(conn)
			}
			//将uniqIds归还给内存池
			objPool.UniqIdSlice.Put(uniqIds)
		}
	}
}
