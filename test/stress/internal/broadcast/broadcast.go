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
	"github.com/tidwall/gjson"
	"golang.org/x/time/rate"
	"netsvr/pkg/quit"
	"netsvr/test/pkg/protocol"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsPool"
	"strings"
	"sync"
	"time"
)

var Pool *pool

type pool struct {
	wsPool.Pool
}

func init() {
	Pool = &pool{}
	Pool.P = map[string]*wsClient.Client{}
	Pool.Mux = &sync.RWMutex{}
	if configs.Config.Heartbeat > 0 {
		go Pool.Heartbeat()
	}
}

func (r *pool) AddWebsocket() {
	r.Pool.AddWebsocket(func(ws *wsClient.Client) {
		ws.OnMessage[protocol.RouterBroadcast] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
	})
}

// Send 广播一条数据
func (r *pool) Send(message string) {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterBroadcast, map[string]interface{}{"message": message})
	}
}

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Broadcast.Enable {
		return
	}
	if configs.Config.Broadcast.MessageInterval > 0 {
		go func() {
			tc := time.NewTicker(time.Second * time.Duration(configs.Config.Broadcast.MessageInterval))
			defer tc.Stop()
			message := "我是一条广播信息"
			if configs.Config.Broadcast.MessageLen > 0 {
				message = strings.Repeat("a", configs.Config.Broadcast.MessageLen)
			}
			for {
				select {
				case <-tc.C:
					Pool.Send(message)
				case <-quit.Ctx.Done():
					return
				}
			}
		}()
	}
	for key, step := range configs.Config.Broadcast.Step {
		if step.ConnNum <= 0 {
			continue
		}
		l := rate.NewLimiter(rate.Limit(step.ConnectNum), step.ConnectNum)
		for i := 0; i < step.ConnNum; i++ {
			if err := l.Wait(quit.Ctx); err != nil {
				continue
			}
			Pool.AddWebsocket()
		}
		log.Logger.Info().Msgf("current broadcast online %d", Pool.Len())
		if key < len(configs.Config.Broadcast.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	go func() {
		<-quit.Ctx.Done()
		Pool.Close()
	}()
}
