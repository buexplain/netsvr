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

// Package topic 对网关主题操作进行压测
package topic

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
		ws.OnMessage[protocol.RouterTopicSubscribe] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
		ws.OnMessage[protocol.RouterTopicUnsubscribe] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
		ws.InitTopic()
	})
}

// Subscribe 订阅主题
func (r *pool) Subscribe() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterTopicSubscribe, map[string][]string{"topics": ws.GetSubscribeTopic()})
	}
}

// WaitPublish 待发布的主题列表
func (r *pool) waitPublish() []string {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	topics := make([]string, 0, len(r.P))
	for _, ws = range r.P {
		topics = append(topics, ws.GetTopic())
	}
	return topics
}

// Unsubscribe 取消订阅主题
func (r *pool) Unsubscribe() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterTopicUnsubscribe, map[string][]string{"topics": ws.GetUnsubscribeTopic()})
	}
}

// Publish 发布主题
func (r *pool) Publish(message string) {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterTopicPublish, map[string]any{"topics": []string{ws.GetTopic()}, "message": message})
	}
}

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Topic.Enable {
		return
	}
	go func() {
		tc := time.NewTicker(time.Second * time.Duration(configs.Config.Topic.MessageInterval))
		defer tc.Stop()
		message := "我是一条发布信息"
		if configs.Config.Topic.MessageLen > 0 {
			message = strings.Repeat("a", configs.Config.Topic.MessageLen)
		}
		for {
			select {
			case <-tc.C:
				Pool.Subscribe()
				if rand.Intn(10) > 5 {
					time.Sleep(time.Millisecond * 100)
				}
				Pool.Unsubscribe()
				Pool.Publish(message)
			case <-quit.Ctx.Done():
				return
			}
		}
	}()
	for key, step := range configs.Config.Topic.Step {
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
		log.Logger.Info().Msgf("current topic online %d", Pool.Len())
		if key < len(configs.Config.Topic.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	log.Logger.Info().Str("topics", strings.Join(Pool.waitPublish(), ",")).Msg("发布的主题")
	go func() {
		<-quit.Ctx.Done()
		Pool.Close()
	}()
}
