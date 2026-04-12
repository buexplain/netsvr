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

// connInfoDelete 删除连接的info信息
func connInfoDelete(param []byte) {
	payload := objPool.ConnInfoDelete.Get()
	defer objPool.ConnInfoDelete.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.connInfoDelete failed")
		return
	}
	if payload.UniqId == "" {
		return
	}
	conn := manager.Manager.Get(payload.UniqId)
	session, ok := customer.GetSession(conn)
	if !ok {
		return
	}
	session.Lock()
	defer session.UnLock()
	if session.GetUniqId() == "" {
		//session已经被销毁了，跳过
		return
	}
	//删除主题
	if payload.DelTopic {
		topics := session.PullTopics()
		topic.Topic.DelRelationByMap(topics, conn)
	}
	//删除session
	if payload.DelSession {
		session.SetSession("")
	}
	if payload.DelCustomerId {
		customerId := session.GetCustomerId()
		if customerId != "" {
			binder.Binder.DelRelation(customerId, conn)
			session.SetCustomerId("")
		}
	}
	//有数据，则转发给客户
	if len(payload.Data) > 0 {
		customer.WriteMessage(conn, configs.Config.Customer.SendMessageType, payload.Data)
	}
}
