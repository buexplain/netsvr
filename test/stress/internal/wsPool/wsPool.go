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

package wsPool

import (
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/wsClient"
	"sync"
	"time"
)

type Pool struct {
	P   map[string]*wsClient.Client
	Mux *sync.RWMutex
}

func (r *Pool) Heartbeat() {
	tc := time.NewTicker(time.Second * 30)
	defer tc.Stop()
	for {
		<-tc.C
		r.Mux.RLock()
		for _, ws := range r.P {
			ws.Heartbeat()
		}
		r.Mux.RUnlock()
	}
}

func (r *Pool) Len() int {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	return len(r.P)
}

func (r *Pool) Close() {
	var ws *wsClient.Client
	for {
		r.Mux.RLock()
		for _, ws = range r.P {
			break
		}
		r.Mux.RUnlock()
		if ws == nil {
			break
		}
		ws.Close()
		ws = nil
	}
}

func (r *Pool) AddWebsocket(option func(ws *wsClient.Client)) {
	ws := wsClient.New(configs.Config.CustomerWsAddress)
	if ws == nil {
		return
	}
	option(ws)
	ws.OnClose = func() {
		r.Mux.Lock()
		defer r.Mux.Unlock()
		delete(r.P, ws.LocalUniqId)
	}
	go ws.LoopSend()
	go ws.LoopRead()
	r.Mux.Lock()
	defer r.Mux.Unlock()
	r.P[ws.LocalUniqId] = ws
}
