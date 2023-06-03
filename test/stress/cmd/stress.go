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
	"fmt"
	"github.com/olekukonko/tablewriter"
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
		log.Logger.Info().Msgf("current online %d",
			silent.Metrics.Online.Count()+
				singleCast.Metrics.Online.Count()+
				multicast.Metrics.Online.Count()+
				broadcast.Metrics.Online.Count()+
				topic.Metrics.Online.Count()+
				sign.Metrics.Online.Count()+0,
		)
		go func() {
			time.Sleep(time.Second * time.Duration(configs.Config.Suspend))
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"模块", "连接数", "发送字节", "接收字节"})
			table.Append([]string{
				"silent",
				fmt.Sprintf("%d", silent.Metrics.Online.Count()),
				fmt.Sprintf("%d", silent.Metrics.Send.Count()),
				fmt.Sprintf("%d", silent.Metrics.Receive.Count()),
			})
			table.Append([]string{
				"singleCast",
				fmt.Sprintf("%d", singleCast.Metrics.Online.Count()),
				fmt.Sprintf("%d", singleCast.Metrics.Send.Count()),
				fmt.Sprintf("%d", singleCast.Metrics.Receive.Count()),
			})
			table.Append([]string{
				"multicast",
				fmt.Sprintf("%d", multicast.Metrics.Online.Count()),
				fmt.Sprintf("%d", multicast.Metrics.Send.Count()),
				fmt.Sprintf("%d", multicast.Metrics.Receive.Count()),
			})
			table.Append([]string{
				"broadcast",
				fmt.Sprintf("%d", broadcast.Metrics.Online.Count()),
				fmt.Sprintf("%d", broadcast.Metrics.Send.Count()),
				fmt.Sprintf("%d", broadcast.Metrics.Receive.Count()),
			})
			table.Append([]string{
				"topic",
				fmt.Sprintf("%d", topic.Metrics.Online.Count()),
				fmt.Sprintf("%d", topic.Metrics.Send.Count()),
				fmt.Sprintf("%d", topic.Metrics.Receive.Count()),
			})
			table.Append([]string{
				"sign",
				fmt.Sprintf("%d", sign.Metrics.Online.Count()),
				fmt.Sprintf("%d", sign.Metrics.Send.Count()),
				fmt.Sprintf("%d", sign.Metrics.Receive.Count()),
			})
			table.Append([]string{
				"总计",
				fmt.Sprintf("%d",
					silent.Metrics.Online.Count()+
						singleCast.Metrics.Online.Count()+
						multicast.Metrics.Online.Count()+
						broadcast.Metrics.Online.Count()+
						topic.Metrics.Online.Count()+
						sign.Metrics.Online.Count()+0,
				),
				fmt.Sprintf("%d",
					silent.Metrics.Send.Count()+
						singleCast.Metrics.Send.Count()+
						multicast.Metrics.Send.Count()+
						broadcast.Metrics.Send.Count()+
						topic.Metrics.Send.Count()+
						sign.Metrics.Send.Count()+0,
				),
				fmt.Sprintf("%d",
					silent.Metrics.Receive.Count()+
						singleCast.Metrics.Receive.Count()+
						multicast.Metrics.Receive.Count()+
						broadcast.Metrics.Receive.Count()+
						topic.Metrics.Receive.Count()+
						sign.Metrics.Receive.Count()+0,
				),
			})
			table.Render()
			quit.Execute("压测完毕")
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
