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
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/objPool"
	"netsvr/internal/utils"
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
	if session.IsClosed() {
		return
	}
	session.MuxLock()
	if session.IsClosed() {
		session.MuxUnLock()
		return
	}
	//记录下老的uniqId
	previousUniqId := payload.UniqId
	//在高并发下，这个payload.UniqId不一定是manager.Manager.Get时候的，所以一定要重新再从session里面拿出来，保持一致，否则接下来的逻辑会导致连接泄漏
	payload.UniqId = session.GetUniqId()
	//删除主题
	if payload.DelTopic {
		topics := session.PullTopics()
		topic.Topic.DelByMap(topics, payload.UniqId, previousUniqId)
	}
	//删除session
	if payload.DelSession {
		session.SetSession("")
	}
	//删除uniqId
	if payload.DelUniqId {
		//生成一个新的uniqId
		newUniqId := utils.UniqId()
		//处理连接管理器中的关系
		manager.Manager.Del(payload.UniqId)
		manager.Manager.Set(newUniqId, conn)
		//处理主题管理器中的关系
		topics := session.SetUniqIdAndGetTopics(newUniqId)
		//删除旧关系，构建新关系
		topic.Topic.DelBySlice(topics, payload.UniqId, previousUniqId)
		topic.Topic.SetBySlice(topics, newUniqId)
	}
	session.MuxUnLock()
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
