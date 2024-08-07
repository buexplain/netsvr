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

// Package singleCastBulk 对网关根据uniqId批量单播的功能进行压测
package singleCastBulk

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
	if !configs.Config.SingleCastBulk.Enable {
		return
	}
	log.Logger.Info().Msgf("singleCastBulk running")
	if configs.Config.SingleCastBulk.SendInterval <= 0 {
		log.Logger.Error().Msg("配置 Config.SingleCastBulk.SendInterval 必须是个大于0的值")
		return
	}
	if configs.Config.SingleCastBulk.UniqIdNum <= 0 {
		log.Logger.Error().Msg("配置 Config.SingleCastBulk.UniqIdNum 必须是个大于0的值")
		return
	}
	message := "我是一条按uniqId的批量单播的信息"
	if configs.Config.SingleCastBulk.MessageLen > 0 {
		message = strings.Repeat("s", configs.Config.SingleCastBulk.MessageLen)
	}
	for key, step := range configs.Config.SingleCastBulk.Step {
		metrics := wsMetrics.New("singleCastBulk", key+1)
		utils.Concurrency(step.ConnNum, step.ConnectNum, func() {
			ws := wsClient.New(configs.Config.CustomerWsAddress, metrics, func(ws *wsClient.Client) {
				ws.OnMessage = nil
			})
			if ws == nil {
				return
			}
			collect.Add(ws)
			uniqIds := collect.RandomGetUniqIds(configs.Config.SingleCastBulk.UniqIdNum)
			wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.SingleCastBulk.SendInterval), func() {
				if len(uniqIds) < configs.Config.SingleCastBulk.UniqIdNum {
					uniqIds = collect.RandomGetUniqIds(configs.Config.SingleCastBulk.UniqIdNum)
				}
				data := make([]string, 0, len(uniqIds))
				for i := len(uniqIds); i > 0; i-- {
					data = append(data, message)
				}
				ws.Send(protocol.RouterSingleCastBulk, map[string]interface{}{"message": data, "uniqIds": uniqIds})
			})
		})
		metrics.RecordConnectOK()
		log.Logger.Info().Msgf("singleCastBulk current step %d online %d", metrics.Step, metrics.Online.Count())
		if key < len(configs.Config.SingleCastBulk.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	log.Logger.Info().Msgf("singleCastBulk current online %d", wsMetrics.Collect.CountByName("singleCastBulk"))
}
