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
	"golang.org/x/time/rate"
	"netsvr/pkg/quit"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsMetrics"
	"sync"
	"time"
)

var Metrics *wsMetrics.WsStatus

func init() {
	Metrics = wsMetrics.New()
}

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Silent.Enable {
		return
	}
	l := rate.NewLimiter(rate.Limit(1), 1)
	for key, step := range configs.Config.Silent.Step {
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
				wsClient.New(configs.Config.CustomerWsAddress, Metrics, func(ws *wsClient.Client) {
					ws.OnMessage = nil
				})
			}
		}
		if key < len(configs.Config.Silent.Step)-1 {
			log.Logger.Info().Msgf("current silent online %d", Metrics.Online.Count())
			if step.Suspend > 0 {
				time.Sleep(time.Duration(step.Suspend) * time.Second)
			}
		}
	}
	log.Logger.Info().Msgf("current silent online %d", Metrics.Online.Count())
}
