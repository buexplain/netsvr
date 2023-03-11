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

// Package multicast 对网关组播进行压测
package multicast

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
		ws.OnMessage[protocol.RouterMulticastForUniqId] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
	})
}

// Send 组播一条数据
func (r *pool) Send(message string, uniqIdNum int) {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	uniqIds := make([]string, 0, uniqIdNum)
	replaceIndex := 0
	for _, ws = range r.P {
		if len(uniqIds) < uniqIdNum {
			uniqIds = append(uniqIds, ws.UniqId)
		} else {
			uniqIds[replaceIndex] = ws.UniqId
			replaceIndex++
			if replaceIndex == uniqIdNum {
				replaceIndex = 0
			}
		}
		ws.Send(protocol.RouterMulticastForUniqId, map[string]interface{}{"message": message, "uniqIds": uniqIds})
	}
}

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Multicast.Enable {
		return
	}
	if configs.Config.Multicast.MessageInterval > 0 && configs.Config.Multicast.UniqIdNum > 0 {
		go func() {
			tc := time.NewTicker(time.Second * time.Duration(configs.Config.Multicast.MessageInterval))
			defer tc.Stop()
			message := "我是一条按uniqId组播的信息"
			if configs.Config.Multicast.MessageLen > 0 {
				message = strings.Repeat("a", configs.Config.Multicast.MessageLen)
			}
			for {
				select {
				case <-tc.C:
					Pool.Send(message, configs.Config.Multicast.UniqIdNum)
				case <-quit.Ctx.Done():
					return
				}
			}
		}()
	}
	for key, step := range configs.Config.Multicast.Step {
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
		log.Logger.Info().Msgf("current multicast online %d", Pool.Len())
		if key < len(configs.Config.Multicast.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	go func() {
		<-quit.Ctx.Done()
		Pool.Close()
	}()
}
