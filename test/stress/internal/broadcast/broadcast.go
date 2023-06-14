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

// Package broadcast 对网关广播进行压测
package broadcast

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
	if !configs.Config.Broadcast.Enable {
		return
	}
	log.Logger.Info().Msgf("broadcast running")
	if configs.Config.Broadcast.MessageInterval <= 0 {
		log.Logger.Error().Msg("配置 Config.Broadcast.MessageInterval 必须是个大于0的值")
		return
	}
	message := "我是一条广播信息"
	if configs.Config.Broadcast.MessageLen > 0 {
		message = strings.Repeat("b", configs.Config.Broadcast.MessageLen)
	}
	data := map[string]interface{}{"message": message}
	for key, step := range configs.Config.Broadcast.Step {
		metrics := wsMetrics.New("broadcast", key+1)
		utils.Concurrency(step.ConnNum, step.ConnectNum, func() {
			ws := wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
				ws.OnMessage = nil
			})
			if ws != nil {
				wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.Broadcast.MessageInterval), func() {
					ws.Send(protocol.RouterBroadcast, data)
				})
			}
		})
		metrics.RecordConnectOK()
		log.Logger.Info().Msgf("broadcast current step %d online %d", metrics.Step, metrics.Online.Count())
		if key < len(configs.Config.Broadcast.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	log.Logger.Info().Msgf("broadcast current online %d", wsMetrics.Collect.CountByName("broadcast"))
}
