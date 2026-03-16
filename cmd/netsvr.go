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

package main

import (
	"github.com/google/gops/agent"
	_ "net/http/pprof"
	"netsvr/configs"
	"netsvr/internal/customer"
	"netsvr/internal/log"
	"netsvr/internal/task"
	"netsvr/internal/worker"
	"netsvr/pkg/quit"
	"os"
	"time"
)

func main() {
	pprof()
	if configs.Config.Autobahn {
		go customer.StartAutobahn()
	} else {
		go customer.Start()
	}
	go worker.Start()
	go task.Start()
	select {
	case <-quit.ClosedCh:
		//及时打印关闭进程的日志，避免使用者认为进程无反应，直接强杀进程
		log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("Start shutting down the netsvr process")
		//通知所有协程开始退出
		quit.Cancel()
		if configs.Config.ShutdownWaitTime > 0 {
			go func() {
				time.Sleep(configs.Config.ShutdownWaitTime)
				log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("Forced shutdown of netsvr process successfully")
				os.Exit(1)
			}()
		}
		//等待协程退出
		quit.Wg.Wait()
		//关闭worker服务器
		worker.Shutdown()
		task.Shutdown()
		//关闭customer服务器
		customer.Shutdown()
		log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("Shutdown the netsvr process successfully")
		os.Exit(0)
	}
}

func pprof() {
	if configs.Config.PprofListenAddress == "" {
		return
	}
	go func() {
		defer func() {
			_ = recover()
		}()
		log.Logger.Info().Int("pid", os.Getpid()).Msgf("gops start, eg: gops memstats %s", configs.Config.PprofListenAddress)
		if err := agent.Listen(agent.Options{
			Addr:                   configs.Config.PprofListenAddress,
			ReuseSocketAddrAndPort: true,
		}); err != nil {
			log.Logger.Error().Err(err)
		}
	}()
}
