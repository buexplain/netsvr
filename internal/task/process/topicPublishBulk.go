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
	"netsvr/configs"
	"netsvr/internal/customer"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/objPool"

	"google.golang.org/protobuf/proto"
)

// topicPublishBulk 批量发布
func topicPublishBulk(param []byte) {
	payload := objPool.TopicPublishBulk.Get()
	defer objPool.TopicPublishBulk.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.topicPublishBulk failed")
		return
	}
	//迭代每一项：本项内每一条数据按顺序发布给本项内每一个主题
	for _, item := range payload.Items {
		for _, data := range item.GetData() {
			//判断数据是否有效
			if len(data) == 0 {
				continue
			}
			msg := customer.FrameObjPool.Get(configs.Config.Customer.SendMessageType, data)
			for _, currentTopic := range item.GetTopics() {
				//获得当前主题下的所有连接
				connList := topic.Topic.GetConnListByTopic(currentTopic, objPool.ConnSlice)
				if connList == nil {
					continue
				}
				connListAlias := *connList //搞个别名，避免循环中解指针，提高性能
				for _, conn := range connListAlias {
					msg.WriteTo(conn)
				}
				//将connList归还给内存池
				objPool.ConnSlice.Put(connList)
			}
			customer.FrameObjPool.Put(msg)
		}
	}
}
