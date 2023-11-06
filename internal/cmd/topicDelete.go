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
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	customerTopic "netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// TopicDelete 删除主题
func TopicDelete(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.TopicDelete.Get()
	defer objPool.TopicDelete.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicDelete failed")
		return
	}
	//topic --> []uniqId
	arr := customerTopic.Topic.PullAndReturnUniqIds(payload.Topics)
	if len(arr) == 0 {
		return
	}
	//只做删除，不需要通知到客户
	if len(payload.Data) == 0 {
		for topic, uniqIds := range arr {
			for uniqId := range uniqIds {
				conn := customerManager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				if session, ok := conn.SessionWithLock().(*info.Info); ok {
					_ = session.UnsubscribeTopic(topic)
				}
			}
		}
		return
	}
	//需要发送给客户数据，注意，只能发送一次
	isSend := false
	//先清空整个payload.Topics，用于记录已经处理过的topic
	payload.Topics = payload.Topics[:0]
	for topic, uniqIds := range arr {
		for uniqId := range uniqIds {
			conn := customerManager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			session, ok := conn.SessionWithLock().(*info.Info)
			if !ok {
				continue
			}
			if !session.UnsubscribeTopic(topic) {
				continue
			}
			//取消订阅成功，判断是否已经发送过数据
			isSend = false
			for _, topic = range payload.Topics {
				//迭代每一个已经处理的topic，判断每个已经处理topic是否包含当前循环的uniqId
				if _, isSend = arr[topic][uniqId]; isSend {
					break
				}
			}
			//没有发送过数据，则发送数据
			if !isSend {
				if err := conn.WriteMessage(configs.Config.Customer.SendMessageType, payload.Data); err == nil {
					metrics.Registry[metrics.ItemCustomerWriteNumber].Meter.Mark(1)
					metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(payload.Data)))
				} else {
					_ = conn.Close()
				}
			}
		}
		//处理完毕的topic，记录起来
		payload.Topics = append(payload.Topics, topic)
	}
}
