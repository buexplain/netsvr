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
	"netsvr/internal/customer/binder"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
)

// connInfoUpdate 更新连接的info信息
func connInfoUpdate(param []byte) {
	payload := objPool.ConnInfoUpdate.Get()
	defer objPool.ConnInfoUpdate.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.connInfoUpdate failed")
		return
	}
	if payload.UniqId == "" {
		return
	}
	conn := manager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	conn.Lock()
	defer conn.UnLock()
	if conn.IsClosedOnSafe() {
		return
	}
	//设置session
	if payload.NewSession != "" {
		conn.SetSession(payload.NewSession)
	}
	//设置customerId
	if payload.NewCustomerId != "" {
		oldCustomerId := conn.GetCustomerId()
		if oldCustomerId != "" {
			//删除旧关系
			binder.Binder.DelRelation(oldCustomerId, conn)
		}
		//设置新关系
		binder.Binder.SetRelation(payload.NewCustomerId, conn)
		conn.SetCustomerId(payload.NewCustomerId)
	}
	//设置主题
	if len(payload.NewTopics) > 0 {
		//清空旧主题
		topics := conn.PullTopics()
		//删除旧主题的关系
		topic.Topic.DelRelationByMap(topics, conn)
		//订阅新主题
		conn.SubscribeTopics(payload.NewTopics)
		//构建新主题的关系
		topic.Topic.SetRelation(payload.NewTopics, conn)
	}
	//有数据，则转发给客户
	if len(payload.Data) > 0 {
		customer.WriteMessage(conn, configs.Config.Customer.SendMessageType, payload.Data)
	}
}
