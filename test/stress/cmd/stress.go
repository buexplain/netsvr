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
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/multicast"
	"netsvr/test/stress/internal/sign"
	"netsvr/test/stress/internal/silent"
	"netsvr/test/stress/internal/singleCast"
	"netsvr/test/stress/internal/topic"
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
			go sign.Run(wg)
			wg.Add(1)
			go singleCast.Run(wg)
			wg.Add(1)
			go multicast.Run(wg)
			wg.Add(1)
			go topic.Run(wg)
			wg.Add(1)
			go broadcast.Run(wg)
		} else {
			silent.Run(nil)
			sign.Run(nil)
			singleCast.Run(nil)
			multicast.Run(nil)
			topic.Run(nil)
			broadcast.Run(nil)
		}
		wg.Wait()
		online := silent.Pool.Len()
		online += sign.Pool.Len()
		online += singleCast.Pool.Len()
		online += multicast.Pool.Len()
		online += broadcast.Pool.Len()
		online += topic.Pool.Len()
		log.Logger.Info().Msgf("current online %d", online)
		go func() {
			time.Sleep(time.Second * time.Duration(configs.Config.Suspend))
			quit.Execute("压测完毕")
		}()
	}()
	select {
	case <-quit.ClosedCh:
		quit.Cancel()
		quit.Wg.Wait()
		log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("Close the stress process successfully")
		os.Exit(0)
	}
}
