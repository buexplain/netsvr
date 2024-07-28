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

// Package limit 限流模块
// 限制客户消息的转发速度
package limit

import (
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v3/netsvr"
	"golang.org/x/time/rate"
	"netsvr/configs"
)

type limiter interface {
	Allow() bool
	Limit() rate.Limit
	SetLimit(newLimit rate.Limit)
}

// 空壳子限流器
type nilLimit struct {
}

func (nilLimit) Allow() bool {
	return true
}

func (nilLimit) Limit() rate.Limit {
	return 0
}
func (nilLimit) SetLimit(_ rate.Limit) {
}

type manager map[netsvrProtocol.Event]limiter

func (r manager) Allow(event netsvrProtocol.Event) bool {
	return r[event].Allow()
}

// Update 更新限流器的并发设定
func (r manager) Update(payload *netsvrProtocol.LimitReq) {
	//更新连接打开事件的限流器
	if payload.OnOpen > 0 && rate.Limit(payload.OnOpen) != Manager[netsvrProtocol.Event_OnOpen].Limit() {
		Manager[netsvrProtocol.Event_OnOpen].SetLimit(rate.Limit(payload.OnOpen))
	}
	//更新消息事件的限流器
	if payload.OnMessage > 0 && rate.Limit(payload.OnMessage) != Manager[netsvrProtocol.Event_OnMessage].Limit() {
		Manager[netsvrProtocol.Event_OnMessage].SetLimit(rate.Limit(payload.OnMessage))
	}
}

func (r manager) Get(payload *netsvrProtocol.LimitResp) {
	payload.OnOpen = int32(Manager[netsvrProtocol.Event_OnOpen].Limit())
	payload.OnMessage = int32(Manager[netsvrProtocol.Event_OnMessage].Limit())
}

var Manager manager

func init() {
	Manager = make(manager)
	//初始化连接打开事件的限流器
	if configs.Config.Limit.OnOpen > 0 {
		Manager[netsvrProtocol.Event_OnOpen] = rate.NewLimiter(rate.Limit(configs.Config.Limit.OnOpen), int(configs.Config.Limit.OnOpen))
	} else {
		Manager[netsvrProtocol.Event_OnOpen] = nilLimit{}
	}
	//初始化消息事件的限流器
	if configs.Config.Limit.OnMessage > 0 {
		Manager[netsvrProtocol.Event_OnMessage] = rate.NewLimiter(rate.Limit(configs.Config.Limit.OnMessage), int(configs.Config.Limit.OnMessage))
	} else {
		Manager[netsvrProtocol.Event_OnMessage] = nilLimit{}
	}
}
