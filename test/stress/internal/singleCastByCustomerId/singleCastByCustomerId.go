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

// Package singleCastByCustomerId 对网关根据customerId单播的功能进行压测
package singleCastByCustomerId

import (
	"github.com/tidwall/gjson"
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
	if !configs.Config.SingleCastByCustomerId.Enable {
		return
	}
	log.Logger.Info().Msgf("singleCastByCustomerId running")
	if configs.Config.SingleCastByCustomerId.SendInterval <= 0 {
		log.Logger.Error().Msg("配置 Config.SingleCastByCustomerId.SendInterval 必须是个大于0的值")
		return
	}
	message := "我是一条按customerId的单播的信息"
	if configs.Config.SingleCastByCustomerId.MessageLen > 0 {
		message = strings.Repeat("s", configs.Config.SingleCastByCustomerId.MessageLen)
	}
	for key, step := range configs.Config.SingleCastByCustomerId.Step {
		metrics := wsMetrics.New("singleCastByCustomerId", key+1)
		utils.Concurrency(step.ConnNum, step.ConnectNum, func() {
			ws := wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
				ws.Send(protocol.RouterSignInForForge, nil)
				ws.OnMessage = map[protocol.Cmd]func(payload gjson.Result){
					protocol.RouterSignInForForge: func(payload gjson.Result) {
						ws.SetCustomerId(payload.Get("data.id").String())
					},
					protocol.Placeholder: func(payload gjson.Result) {
					},
				}
			})
			if ws == nil {
				return
			}
			wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.SingleCastByCustomerId.SendInterval), func() {
				customerId := ws.GetCustomerId()
				if customerId != "" {
					ws.Send(protocol.RouterSingleCastByCustomerId, map[string]string{"message": message, "customerId": customerId})
				}
			})
		})
		metrics.RecordConnectOK()
		log.Logger.Info().Msgf("singleCastByCustomerId current step %d online %d", metrics.Step, metrics.Online.Count())
		if key < len(configs.Config.SingleCastByCustomerId.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	log.Logger.Info().Msgf("singleCastByCustomerId current online %d", wsMetrics.Collect.CountByName("singleCastByCustomerId"))
}
