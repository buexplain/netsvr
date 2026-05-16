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
	testUtils "netsvr/test/pkg/utils"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/utils"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsCollect"
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
	if configs.Config.GroupChat.Limit == 0 {
		log.Logger.Error().Msg("配置 Config.GroupChat.Limit 必须是个有效的值")
		return
	}
	sendLimitB := 1
	if configs.Config.GroupChat.Limit > 0 {
		sendLimitB = int(configs.Config.GroupChat.Limit * 1.6)
	}
	message := "我是一条群聊信息"
	if configs.Config.Topic.Publish.MessageLen > 0 {
		message = strings.Repeat("g", configs.Config.GroupChat.MessageLen)
	}
	signInParam := map[string]uint{"sessionLen": configs.Config.GroupChat.SessionLen}
	//循环构建群
	for groupId := 1; groupId <= configs.Config.GroupChat.GroupNum; groupId++ {
		groupName := "groupChat-" + strconv.Itoa(groupId) //统计的名字
		groupNameTopic := groupName + testUtils.GlobalId  //真实的群名字，加上全局随机id，确保多个压测程序订阅的topic不冲突
		groupMessage := map[string]any{"topics": []string{groupNameTopic}, "message": message}
		collect := wsCollect.New()                                                           //收集群成员连接
		sendLimit := rate.NewLimiter(rate.Limit(configs.Config.GroupChat.Limit), sendLimitB) //发送限流
		//启动多个协程去发送群消息
		for i := 0; i < sendLimitB; i++ {
			go func(collect *wsCollect.Collect, sendLimit *rate.Limiter, groupMessage map[string]any) {
				for {
					if err := sendLimit.Wait(quit.Ctx); err != nil {
						return
					}
					if ws := collect.RandomGet(); ws != nil {
						ws.Send(protocol.RouterTopicPublish, groupMessage)
					}
				}
			}(collect, sendLimit, groupMessage)
		}
		for key, step := range configs.Config.GroupChat.Step {
			metrics := wsMetrics.New(groupName, key+1)
			utils.Concurrency(step.ConnNum, step.ConnectNum, func() {
				ws := wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
					//模拟登录
					ws.Send(protocol.RouterSignInForForge, signInParam)
					ws.OnMessage = map[protocol.Cmd]func(payload gjson.Result){
						//登录成功
						protocol.RouterSignInForForge: func(payload gjson.Result) {
							//发起订阅
							ws.Send(protocol.RouterTopicSubscribe, map[string][]string{"topics": {
								groupNameTopic,
							}})
						},
						//去掉其它命令发来的信息
						protocol.Placeholder: func(_ gjson.Result) {
						},
					}
				})
				if ws == nil {
					return
				}
				collect.Add(ws)
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
