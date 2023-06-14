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

// Package topic 对网关主题操作进行压测
package topic

import (
	"netsvr/test/pkg/protocol"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/utils"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsMetrics"
	"netsvr/test/stress/internal/wsTimer"
	"strings"
	"sync"
	"time"
)

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Topic.Enable {
		return
	}
	log.Logger.Info().Msgf("topic running")
	if configs.Config.Topic.AlternateTopicNum <= 0 {
		log.Logger.Error().Msg("配置 Config.Topic.AlternateTopicNum 必须是个大于0的值")
		return
	}
	if configs.Config.Topic.AlternateTopicLen <= 0 {
		log.Logger.Error().Msg("配置 Config.Topic.AlternateTopicLen 必须是个大于0的值")
		return
	}
	message := "我是一条订阅信息"
	if configs.Config.Topic.Publish.MessageLen > 0 {
		message = strings.Repeat("t", configs.Config.Topic.Publish.MessageLen)
	}
	for key, step := range configs.Config.Topic.Step {
		metrics := wsMetrics.New("topic", key+1)
		utils.Concurrency(step.ConnNum, step.ConnectNum, func() {
			ws := wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
				ws.InitTopic(configs.Config.Topic.AlternateTopicNum, configs.Config.Topic.AlternateTopicLen)
				ws.OnMessage = nil
			})
			if ws == nil {
				return
			}
			//处理订阅
			if configs.Config.Topic.Subscribe.ModeSecond < 0 {
				configs.Config.Topic.Subscribe.ModeSecond = 1
			}
			if configs.Config.Topic.Subscribe.TopicNum < 0 {
				configs.Config.Topic.Subscribe.TopicNum = 1
			}
			if configs.Config.Topic.Subscribe.Mode == configs.ModeAfter {
				wsTimer.WsTimer.AfterFunc(time.Second*time.Duration(configs.Config.Topic.Subscribe.ModeSecond), func() {
					ws.Send(protocol.RouterTopicSubscribe, map[string][]string{"topics": ws.GetSubscribeTopic(configs.Config.Topic.Subscribe.TopicNum)})
				})
			} else if configs.Config.Topic.Subscribe.Mode == configs.ModeSchedule {
				wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.Topic.Subscribe.ModeSecond), func() {
					ws.Send(protocol.RouterTopicSubscribe, map[string][]string{"topics": ws.GetSubscribeTopic(configs.Config.Topic.Subscribe.TopicNum)})
				})
			}
			//处理取消订阅
			if configs.Config.Topic.Unsubscribe.ModeSecond < 0 {
				configs.Config.Topic.Unsubscribe.ModeSecond = 1
			}
			if configs.Config.Topic.Unsubscribe.TopicNum < 0 {
				configs.Config.Topic.Unsubscribe.TopicNum = 1
			}
			if configs.Config.Topic.Unsubscribe.Mode == configs.ModeAfter {
				wsTimer.WsTimer.AfterFunc(time.Second*time.Duration(configs.Config.Topic.Unsubscribe.ModeSecond), func() {
					ws.Send(protocol.RouterTopicUnsubscribe, map[string][]string{"topics": ws.GetUnsubscribeTopic(configs.Config.Topic.Unsubscribe.TopicNum)})
				})
			} else if configs.Config.Topic.Unsubscribe.Mode == configs.ModeSchedule {
				wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.Topic.Unsubscribe.ModeSecond), func() {
					ws.Send(protocol.RouterTopicUnsubscribe, map[string][]string{"topics": ws.GetUnsubscribeTopic(configs.Config.Topic.Unsubscribe.TopicNum)})
				})
			}
			//处理发布
			if configs.Config.Topic.Publish.ModeSecond < 0 {
				configs.Config.Topic.Publish.ModeSecond = 1
			}
			if configs.Config.Topic.Publish.TopicNum < 0 {
				configs.Config.Topic.Publish.TopicNum = 1
			}
			if configs.Config.Topic.Publish.Mode == configs.ModeAfter {
				wsTimer.WsTimer.AfterFunc(time.Second*time.Duration(configs.Config.Topic.Publish.ModeSecond), func() {
					ws.Send(protocol.RouterTopicPublish, map[string]any{"topics": ws.GetPublishTopic(configs.Config.Topic.Publish.TopicNum), "message": message})
				})
			} else if configs.Config.Topic.Publish.Mode == configs.ModeSchedule {
				wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.Topic.Publish.ModeSecond), func() {
					ws.Send(protocol.RouterTopicPublish, map[string]any{"topics": ws.GetPublishTopic(configs.Config.Topic.Publish.TopicNum), "message": message})
				})
			}
		})
		metrics.RecordConnectOK()
		log.Logger.Info().Msgf("topic current step %d online %d", metrics.Step, metrics.Online.Count())
		if key < len(configs.Config.Topic.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	log.Logger.Info().Msgf("topic current online %d", wsMetrics.Collect.CountByName("topic"))
}
