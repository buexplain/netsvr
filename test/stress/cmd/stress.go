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

// 这个是压测模块
package main

import (
	"netsvr/pkg/quit"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/broadcast"
	"netsvr/test/stress/internal/groupChat"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/multicast"
	"netsvr/test/stress/internal/multicastByCustomerId"
	"netsvr/test/stress/internal/sign"
	"netsvr/test/stress/internal/silent"
	"netsvr/test/stress/internal/singleCast"
	"netsvr/test/stress/internal/singleCastBulk"
	"netsvr/test/stress/internal/singleCastBulkByCustomerId"
	"netsvr/test/stress/internal/singleCastByCustomerId"
	"netsvr/test/stress/internal/topic"
	"netsvr/test/stress/internal/wsMetrics"
	"os"
	"sync"
	"time"
)

func main() {
	go func() {
		wg := &sync.WaitGroup{}
		if configs.Config.Concurrent {
			wg.Add(1)
			go silent.Run(wg)
			wg.Add(1)
			go groupChat.Run(wg)
			wg.Add(1)
			go sign.Run(wg)
			wg.Add(1)
			go singleCast.Run(wg)
			wg.Add(1)
			go singleCastByCustomerId.Run(wg)
			wg.Add(1)
			go singleCastBulk.Run(wg)
			wg.Add(1)
			go singleCastBulkByCustomerId.Run(wg)
			wg.Add(1)
			go multicast.Run(wg)
			wg.Add(1)
			go multicastByCustomerId.Run(wg)
			wg.Add(1)
			go topic.Run(wg)
			wg.Add(1)
			go broadcast.Run(wg)
		} else {
			silent.Run(nil)
			groupChat.Run(nil)
			sign.Run(nil)
			singleCast.Run(nil)
			singleCastByCustomerId.Run(nil)
			singleCastBulk.Run(nil)
			singleCastBulkByCustomerId.Run(nil)
			multicast.Run(nil)
			multicastByCustomerId.Run(nil)
			topic.Run(nil)
			broadcast.Run(nil)
		}
		wg.Wait()
		log.Logger.Info().Msgf("all connections established, current online %d", wsMetrics.Collect.Count())
		go func() {
			if wsMetrics.Collect.Count() > 0 {
				time.Sleep(time.Second * time.Duration(configs.Config.Suspend))
			}
			ret := wsMetrics.Collect.ToTable()
			_, _ = ret.WriteTo(os.Stdout)
			quit.Execute("stress ok")
		}()
	}()
	select {
	case <-quit.ClosedCh:
		log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("Start shutting down the stress process")
		quit.Cancel()
		quit.Wg.Wait()
		log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("Shutdown of netsvr stress successfully")
		os.Exit(0)
	}
}
