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
	"golang.org/x/time/rate"
	"netsvr/pkg/quit"
	"netsvr/test/pkg/protocol"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/utils"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsCollect"
	"netsvr/test/stress/internal/wsMetrics"
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
	if !configs.Config.Broadcast.Enable {
		return
	}
	log.Logger.Info().Msgf("broadcast running")
	if configs.Config.Broadcast.Limit == 0 {
		log.Logger.Error().Msg("配置 Config.Broadcast.Limit 必须是个有效的值")
		return
	}
	sendLimitB := 1
	if configs.Config.Broadcast.Limit > 0 {
		sendLimitB = max(int(configs.Config.Broadcast.Limit*1.6), 1)
	}
	message := "我是一条广播信息"
	if configs.Config.Broadcast.MessageLen > 0 {
		message = strings.Repeat("b", configs.Config.Broadcast.MessageLen)
	}
	data := map[string]interface{}{"message": message}
	sendLimit := rate.NewLimiter(rate.Limit(configs.Config.Broadcast.Limit), sendLimitB)
	for i := 0; i < sendLimitB; i++ {
		go func() {
			for {
				if err := sendLimit.Wait(quit.Ctx); err != nil {
					return
				}
				if ws := collect.RandomGet(); ws != nil {
					ws.Send(protocol.RouterBroadcast, data)
				}
			}
		}()
	}
	for key, step := range configs.Config.Broadcast.Step {
		metrics := wsMetrics.New("broadcast", key+1)
		utils.Concurrency(step.ConnNum, step.ConnectNum, func() {
			ws := wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
				ws.OnMessage = nil
			})
			if ws == nil {
				return
			}
			collect.Add(ws)
		})
		metrics.RecordConnectOK()
		log.Logger.Info().Msgf("broadcast current step %d online %d", metrics.Step, metrics.Online.Count())
		if key < len(configs.Config.Broadcast.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	log.Logger.Info().Msgf("broadcast current online %d", wsMetrics.Collect.CountByName("broadcast"))
}
