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
	"github.com/panjf2000/gnet/v2"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/customer"
	customerTopic "netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
)

// topicDelete 删除主题
func topicDelete(param []byte) {
	payload := objPool.TopicDelete.Get()
	defer objPool.TopicDelete.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.topicDelete failed")
		return
	}
	arr := customerTopic.Topic.Del(payload.Topics)
	if len(arr) == 0 {
		return
	}
	//只做删除，不需要通知到客户
	if len(payload.Data) == 0 {
		for topic, connMap := range arr {
			for _, conn := range connMap {
				session, ok := customer.GetSession(conn)
				if ok {
					_ = session.UnsubscribeTopic(topic)
				}
			}
		}
		return
	}
	//需要发送给客户数据，注意，只能发送一次
	msg := customer.NewMessage(configs.Config.Customer.SendMessageType, payload.Data)
	//先统计conn数量
	connCount := 0
	for _, connMap := range arr {
		connCount += len(connMap)
	}
	//创建一个map，用于去重
	unsubscribeConn := make(map[int]gnet.Conn, connCount)
	//遍历所有主题
	for topic, connMap := range arr {
		for _, conn := range connMap {
			session, ok := customer.GetSession(conn)
			if !ok {
				continue
			}
			//取消订阅
			if session.UnsubscribeTopic(topic) {
				unsubscribeConn[conn.Fd()] = conn
			}
		}
	}
	//发送取消订阅消息
	for _, conn := range unsubscribeConn {
		msg.WriteTo(conn)
	}
}
