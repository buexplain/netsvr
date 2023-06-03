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

// Package multicast 对网关组播进行压测
package multicast

import (
	"golang.org/x/time/rate"
	"netsvr/pkg/quit"
	"netsvr/test/pkg/protocol"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsCollect"
	"netsvr/test/stress/internal/wsMetrics"
	"netsvr/test/stress/internal/wsTimer"
	"strings"
	"sync"
	"time"
)

var Metrics *wsMetrics.WsStatus
var collect *wsCollect.Collect

func init() {
	Metrics = wsMetrics.New()
	collect = wsCollect.New()
}

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Multicast.Enable {
		return
	}
	if configs.Config.Multicast.MessageInterval <= 0 {
		log.Logger.Error().Msg("配置 Config.Multicast.MessageInterval 必须是个大于0的值")
		return
	}
	if configs.Config.Multicast.UniqIdNum <= 0 {
		log.Logger.Error().Msg("配置 Config.Multicast.UniqIdNum 必须是个大于0的值")
		return
	}
	message := "我是一条组播信息"
	if configs.Config.Multicast.MessageLen > 0 {
		message = strings.Repeat("m", configs.Config.Multicast.MessageLen)
	}
	l := rate.NewLimiter(rate.Limit(1), 1)
	for key, step := range configs.Config.Multicast.Step {
		if step.ConnNum <= 0 {
			continue
		}
		l.SetLimit(rate.Limit(step.ConnectNum))
		l.SetBurst(step.ConnectNum)
		for i := 0; i < step.ConnNum; i++ {
			if err := l.Wait(quit.Ctx); err != nil {
				return
			}
			select {
			case <-quit.Ctx.Done():
				return
			default:
				ws := wsClient.New(configs.Config.CustomerWsAddress, Metrics, func(ws *wsClient.Client) {
					ws.OnMessage = nil
				})
				if ws != nil {
					collect.Add(ws)
					uniqIds := collect.RandomGetUniqIds(configs.Config.Multicast.UniqIdNum)
					wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.Multicast.MessageInterval), func() {
						if len(uniqIds) < configs.Config.Multicast.UniqIdNum {
							uniqIds = collect.RandomGetUniqIds(configs.Config.Multicast.UniqIdNum)
						}
						ws.Send(protocol.RouterMulticastForUniqId, map[string]interface{}{"message": message, "uniqIds": uniqIds})
					})
				}
			}
		}
		if key < len(configs.Config.Multicast.Step)-1 {
			log.Logger.Info().Msgf("current multicast online %d", Metrics.Online.Count())
			if step.Suspend > 0 {
				time.Sleep(time.Duration(step.Suspend) * time.Second)
			}
		}
	}
	log.Logger.Info().Msgf("current multicast online %d", Metrics.Online.Count())
}
