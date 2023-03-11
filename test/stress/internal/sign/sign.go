/**
* Copyright 2022 buexplain@qq.com
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

// Package sign 对网关登录登出进行压测
package sign

import (
	"github.com/tidwall/gjson"
	"golang.org/x/time/rate"
	"math/rand"
	"netsvr/pkg/quit"
	"netsvr/test/pkg/protocol"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsPool"
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
		ws.OnMessage[protocol.RouterSignInForForge] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
		ws.OnMessage[protocol.RouterSignOutForForge] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
	})
}

// In 登录操作
func (r *pool) In() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterSignInForForge, nil)
	}
}

// Out 退出登录操作
func (r *pool) Out() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterSignOutForForge, nil)
	}
}

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Sign.Enable {
		return
	}
	if configs.Config.Sign.MessageInterval > 0 {
		go func() {
			tc := time.NewTicker(time.Second * time.Duration(configs.Config.Sign.MessageInterval))
			defer tc.Stop()
			for {
				select {
				case <-tc.C:
					Pool.In()
					if rand.Intn(10) > 5 {
						time.Sleep(time.Millisecond * 100)
					}
					Pool.Out()
				case <-quit.Ctx.Done():
					return
				}
			}
		}()
	}
	for key, step := range configs.Config.Sign.Step {
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
		log.Logger.Info().Msgf("current sign online %d", Pool.Len())
		if key < len(configs.Config.Sign.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	go func() {
		<-quit.Ctx.Done()
		Pool.Close()
	}()
}
