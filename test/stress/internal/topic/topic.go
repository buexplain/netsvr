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
	"golang.org/x/time/rate"
	"math/rand"
	"netsvr/pkg/quit"
	"netsvr/test/pkg/protocol"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
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
	if configs.Config.Topic.MessageInterval <= 0 {
		log.Logger.Error().Msg("配置 Config.Topic.MessageInterval 必须是个大于0的值")
		return
	}
	message := "我是一条发布信息"
	if configs.Config.Topic.MessageLen > 0 {
		message = strings.Repeat("t", configs.Config.Topic.MessageLen)
	}
	messageInterval := configs.Config.Topic.MessageInterval * 1000
	l := rate.NewLimiter(rate.Limit(1), 1)
	for key, step := range configs.Config.Topic.Step {
		metrics := wsMetrics.New("topic", key+1)
		if step.ConnNum > 0 && step.ConnectNum > 0 {
			l.SetLimit(rate.Limit(step.ConnectNum))
			l.SetBurst(step.ConnectNum)
		}
		for i := 0; i < step.ConnNum; i++ {
			if err := l.Wait(quit.Ctx); err != nil {
				return
			}
			select {
			case <-quit.Ctx.Done():
				return
			default:
				ws := wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
					ws.InitTopic()
					ws.OnMessage = nil
				})
				if ws == nil {
					continue
				}
				if rand.Intn(10) > 5 {
					//先发订阅指令
					wsTimer.WsTimer.ScheduleFunc(time.Millisecond*time.Duration(messageInterval), func() {
						ws.Send(protocol.RouterTopicSubscribe, map[string][]string{"topics": ws.GetSubscribeTopic()})
					})
					//间隔200毫秒后再发取消订阅指令
					wsTimer.WsTimer.ScheduleFunc(time.Millisecond*time.Duration(messageInterval+200), func() {
						ws.Send(protocol.RouterTopicUnsubscribe, map[string][]string{"topics": ws.GetUnsubscribeTopic()})
					})
					//间隔200毫秒后再发发布指令
					wsTimer.WsTimer.ScheduleFunc(time.Millisecond*time.Duration(messageInterval+400), func() {
						ws.Send(protocol.RouterTopicPublish, map[string]any{"topics": []string{ws.GetTopic()}, "message": message})
					})
				} else {
					//无缝发送订阅、取消订阅、发布指令
					wsTimer.WsTimer.ScheduleFunc(time.Millisecond*time.Duration(messageInterval), func() {
						ws.Send(protocol.RouterTopicSubscribe, map[string][]string{"topics": ws.GetSubscribeTopic()})
						ws.Send(protocol.RouterTopicUnsubscribe, map[string][]string{"topics": ws.GetUnsubscribeTopic()})
						ws.Send(protocol.RouterTopicPublish, map[string]any{"topics": []string{ws.GetTopic()}, "message": message})
					})
				}
			}
		}
		log.Logger.Info().Msgf("current topic step %d online %d", metrics.Step, metrics.Online.Count())
		if key < len(configs.Config.Topic.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
}
