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

// Package multicastByCustomerId 对网关按customerId组播的功能进行压测
package multicastByCustomerId

import (
	"github.com/tidwall/gjson"
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
	if !configs.Config.MulticastByCustomerId.Enable {
		return
	}
	log.Logger.Info().Msgf("multicastByCustomerId running")
	if configs.Config.MulticastByCustomerId.SendInterval <= 0 {
		log.Logger.Error().Msg("配置 Config.MulticastByCustomerId.SendInterval 必须是个大于0的值")
		return
	}
	if configs.Config.MulticastByCustomerId.CustomerIdNum <= 0 {
		log.Logger.Error().Msg("配置 Config.MulticastByCustomerId.CustomerIdNum 必须是个大于0的值")
		return
	}
	message := "我是一条按customerId组播的信息"
	if configs.Config.MulticastByCustomerId.MessageLen > 0 {
		message = strings.Repeat("m", configs.Config.MulticastByCustomerId.MessageLen)
	}
	for key, step := range configs.Config.MulticastByCustomerId.Step {
		metrics := wsMetrics.New("multicastByCustomerId", key+1)
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
			collect.Add(ws)
			customerIds := collect.RandomGetCustomerIds(configs.Config.MulticastByCustomerId.CustomerIdNum)
			wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.MulticastByCustomerId.SendInterval), func() {
				if len(customerIds) < configs.Config.MulticastByCustomerId.CustomerIdNum {
					customerIds = collect.RandomGetUniqIds(configs.Config.MulticastByCustomerId.CustomerIdNum)
				}
				ws.Send(protocol.RouterMulticastByCustomerId, map[string]interface{}{"message": message, "customerIds": customerIds})
			})
		})
		metrics.RecordConnectOK()
		log.Logger.Info().Msgf("multicastByCustomerId current step %d online %d", metrics.Step, metrics.Online.Count())
		if key < len(configs.Config.MulticastByCustomerId.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	log.Logger.Info().Msgf("multicastByCustomerId current online %d", wsMetrics.Collect.CountByName("multicastByCustomerId"))
}
