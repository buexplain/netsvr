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
	"netsvr/internal/customer/binder"
	"netsvr/internal/customer/info"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// ConnInfoDelete 删除连接的info信息
func ConnInfoDelete(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.ConnInfoDelete.Get()
	defer objPool.ConnInfoDelete.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.ConnInfoDelete failed")
		return
	}
	if payload.UniqId == "" {
		return
	}
	conn := manager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	session, ok := conn.SessionWithLock().(*info.Info)
	if !ok {
		return
	}
	session.Lock()
	defer session.UnLock()
	if session.GetUniqId() == "" {
		return
	}
	//删除主题
	if payload.DelTopic {
		topics := session.PullTopics()
		topic.Topic.DelByMap(topics, payload.UniqId)
	}
	//删除session
	if payload.DelSession {
		session.SetSession("")
	}
	if payload.DelCustomerId {
		//如果客户id为空，则没必要去解除绑定关系，因为解除绑定关系需要获取互斥锁
		if session.GetCustomerId() != "" {
			binder.Binder.DelUniqId(payload.UniqId)
		}
		session.SetCustomerId("")
	}
	//有数据，则转发给客户
	if len(payload.Data) > 0 {
		if err := conn.WriteMessage(configs.Config.Customer.SendMessageType, payload.Data); err == nil {
			metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(payload.Data)))
		} else {
			_ = conn.Close()
		}
	}
}
