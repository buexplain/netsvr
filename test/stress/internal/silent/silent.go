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

// Package silent 对网关连接负载进行压测
package silent

import (
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/utils"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsMetrics"
	"sync"
	"time"
)

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Silent.Enable {
		return
	}
	log.Logger.Info().Msgf("silent running")
	for key, step := range configs.Config.Silent.Step {
		metrics := wsMetrics.New("silent", key+1)
		utils.Concurrency(step.ConnNum, step.ConnectNum, func() {
			wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
				ws.OnMessage = nil
			})
		})
		metrics.RecordConnectOK()
		log.Logger.Info().Msgf("silent current step %d online %d", metrics.Step, metrics.Online.Count())
		if key < len(configs.Config.Silent.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	log.Logger.Info().Msgf("silent current online %d", wsMetrics.Collect.CountByName("silent"))
}
