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

// Package multicast 对网关按uniqId组播的功能进行压测
package multicast

import (
	"netsvr/test/pkg/protocol"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/utils"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsCollect"
	"netsvr/test/stress/internal/wsMetrics"
	"netsvr/test/stress/internal/wsTimer"
	"strings"
	"sync"
	"time"
)

var collect *wsCollect.Collect

func init() {
	collect = wsCollect.New()
}

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Multicast.Enable {
		return
	}
	log.Logger.Info().Msgf("multicast running")
	if configs.Config.Multicast.SendInterval <= 0 {
		log.Logger.Error().Msg("配置 Config.Multicast.SendInterval 必须是个大于0的值")
		return
	}
	if configs.Config.Multicast.UniqIdNum <= 0 {
		log.Logger.Error().Msg("配置 Config.Multicast.UniqIdNum 必须是个大于0的值")
		return
	}
	message := "我是一条按uniqId组播的信息"
	if configs.Config.Multicast.MessageLen > 0 {
		message = strings.Repeat("m", configs.Config.Multicast.MessageLen)
	}
	for key, step := range configs.Config.Multicast.Step {
		metrics := wsMetrics.New("multicast", key+1)
		utils.Concurrency(step.ConnNum, step.ConnectNum, func() {
			ws := wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
				ws.OnMessage = nil
			})
			if ws == nil {
				return
			}
			collect.Add(ws)
			uniqIds := collect.RandomGetUniqIds(configs.Config.Multicast.UniqIdNum)
			wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.Multicast.SendInterval), func() {
				if len(uniqIds) < configs.Config.Multicast.UniqIdNum {
					uniqIds = collect.RandomGetUniqIds(configs.Config.Multicast.UniqIdNum)
				}
				ws.Send(protocol.RouterMulticast, map[string]interface{}{"message": message, "uniqIds": uniqIds})
			})
		})
		metrics.RecordConnectOK()
		log.Logger.Info().Msgf("multicast current step %d online %d", metrics.Step, metrics.Online.Count())
		if key < len(configs.Config.Multicast.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	log.Logger.Info().Msgf("multicast current online %d", wsMetrics.Collect.CountByName("multicast"))
}
