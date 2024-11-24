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

// Package groupChat 对网关登录、订阅、群聊进行压测
package groupChat

import (
	"github.com/tidwall/gjson"
	"golang.org/x/time/rate"
	"netsvr/pkg/quit"
	"netsvr/test/pkg/protocol"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/utils"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsMetrics"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.GroupChat.Enable {
		return
	}
	log.Logger.Info().Msgf("groupChat running")
	message := "我是一条群聊信息"
	if configs.Config.Topic.Publish.MessageLen > 0 {
		message = strings.Repeat("g", configs.Config.GroupChat.MessageLen)
	}
	signInParam := map[string]uint{"sessionLen": configs.Config.GroupChat.SessionLen}
	for groupId := 1; groupId <= configs.Config.GroupChat.GroupNum; groupId++ {
		groupName := "groupChat-" + strconv.Itoa(groupId)
		groupSendLimit := rate.NewLimiter(rate.Limit(configs.Config.GroupChat.SendSpeed), configs.Config.GroupChat.SendSpeed)
		groupSenderNum := 0
		groupMessage := map[string]any{"topics": []string{groupName}, "message": message}
		for key, step := range configs.Config.GroupChat.Step {
			metrics := wsMetrics.New(groupName, key+1)
			utils.Concurrency(step.ConnNum, step.ConnectNum, func() {
				wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
					//模拟登录
					ws.Send(protocol.RouterSignInForForge, signInParam)
					ws.OnMessage = map[protocol.Cmd]func(payload gjson.Result){
						//登录成功
						protocol.RouterSignInForForge: func(payload gjson.Result) {
							//发起订阅
							ws.Send(protocol.RouterTopicSubscribe, map[string][]string{"topics": {groupName}})
						},
						//订阅成功
						protocol.RouterTopicSubscribe: func(payload gjson.Result) {
							//模拟群聊
							if groupSenderNum > configs.Config.GroupChat.SendSpeed*2 {
								//没必要搞那么多人去发送消息
								return
							}
							groupSenderNum++
							go func() {
								for {
									if err := groupSendLimit.Wait(quit.Ctx); err == nil {
										ws.Send(protocol.RouterTopicPublish, groupMessage)
									}
								}
							}()
						},
						//模拟群聊成功
						protocol.RouterTopicPublish: func(_ gjson.Result) {
						},
					}
				})
			})
			metrics.RecordConnectOK()
			log.Logger.Info().Msgf("%s current step %d online %d", groupName, metrics.Step, metrics.Online.Count())
			if key < len(configs.Config.GroupChat.Step)-1 && step.Suspend > 0 {
				time.Sleep(time.Duration(step.Suspend) * time.Second)
			}
		}
		log.Logger.Info().Msgf("%s current online %d", groupName, wsMetrics.Collect.CountByName(groupName))
	}
}
