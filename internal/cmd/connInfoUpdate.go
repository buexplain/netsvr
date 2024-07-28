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

// ConnInfoUpdate 更新连接的info信息
func ConnInfoUpdate(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.ConnInfoUpdate.Get()
	defer objPool.ConnInfoUpdate.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.ConnInfoUpdate failed")
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
		//session已经被销毁了，跳过
		return
	}
	//设置session
	if payload.NewSession != "" {
		session.SetSession(payload.NewSession)
	}
	//设置customerId
	if payload.NewCustomerId != "" {
		binder.Binder.Set(payload.UniqId, payload.NewCustomerId)
		session.SetCustomerId(payload.NewCustomerId)
	}
	//设置主题
	if len(payload.NewTopics) > 0 {
		//清空旧主题
		topics := session.PullTopics()
		//删除旧主题的关系
		topic.Topic.DelByMap(topics, payload.UniqId)
		//订阅新主题
		session.SubscribeTopics(payload.NewTopics)
		//构建新主题的关系
		topic.Topic.SetBySlice(payload.NewTopics, payload.UniqId)
	}
	//有数据，则转发给客户
	if len(payload.Data) > 0 {
		if err := conn.WriteMessage(configs.Config.Customer.SendMessageType, payload.Data); err == nil {
			metrics.Registry[metrics.ItemCustomerWriteNumber].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(payload.Data)))
		} else {
			_ = conn.Close()
		}
	}
}
