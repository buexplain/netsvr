/**
* Copyright 2022 buexplain@qq.com
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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/protocol"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/timer"
	workerManager "netsvr/internal/worker/manager"
	"time"
)

// InfoUpdate 更新连接的info信息
func InfoUpdate(param []byte, _ *workerManager.ConnProcessor) {
	payload := &netsvrProtocol.InfoUpdate{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.InfoUpdate failed")
		return
	}
	if payload.UniqId == "" {
		return
	}
	conn := manager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	session, ok := conn.Session().(*info.Info)
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
	//设置uniqId
	if payload.NewUniqId != "" && payload.NewUniqId != payload.UniqId {
		//如果新的uniqId已经存在，则要移除掉所有关系，因为接下来，这个新的uniqId会被作用在新的连接上
		conflictConn := manager.Manager.Get(payload.NewUniqId)
		if conflictConn != nil {
			//判断是否转发数据
			if len(payload.DataAsNewUniqIdExisted) == 0 {
				//无须转达任何数据，直接关闭连接
				_ = conflictConn.Close()
			} else {
				//写入数据，并在一定倒计时后关闭连接
				_ = conflictConn.WriteMessage(websocket.TextMessage, payload.DataAsNewUniqIdExisted)
				timer.Timer.AfterFunc(time.Second*3, func() {
					defer func() {
						_ = recover()
					}()
					_ = conflictConn.Close()
				})
			}
		}
		//处理连接管理器中的关系
		manager.Manager.Del(payload.UniqId)
		manager.Manager.Set(payload.NewUniqId, conn)
		//处理主题管理器中的关系
		if len(payload.NewTopics) > 0 {
			//如果需要设置新的主题，则在这里一并搞定，设置新的uniqId
			topics := session.SetUniqIdAndPUllTopics(payload.NewUniqId)
			//移除旧主题的关系
			topic.Topic.DelByMap(topics, payload.UniqId, previousUniqId)
			//订阅新主题
			session.SubscribeTopics(payload.NewTopics)
			//设置新主题的关系
			topic.Topic.SetBySlice(payload.NewTopics, payload.NewUniqId)
			//清空掉，避免接下来的逻辑重新设置主题
			payload.NewTopics = nil
		} else {
			topics := session.SetUniqIdAndGetTopics(payload.NewUniqId)
			//删除旧关系，构建新关系
			topic.Topic.DelBySlice(topics, payload.UniqId, previousUniqId)
			topic.Topic.SetBySlice(topics, payload.NewUniqId)
		}
	}
	//设置session
	if payload.NewSession != "" {
		session.SetSession(payload.NewSession)
	}
	//设置主题
	if len(payload.NewTopics) > 0 {
		//清空主题
		topics := session.PullTopics()
		//删除关系
		topic.Topic.DelByMap(topics, payload.UniqId, previousUniqId)
		//订阅主题
		session.SubscribeTopics(payload.NewTopics)
		//构建关系
		topic.Topic.SetBySlice(payload.NewTopics, payload.UniqId)
	}
	session.MuxUnLock()
	//有数据，则转发给客户
	if len(payload.Data) > 0 {
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			_ = conn.Close()
		}
	}
}
