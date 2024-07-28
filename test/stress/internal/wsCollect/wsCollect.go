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

package wsCollect

import (
	"netsvr/test/stress/internal/wsClient"
	"sync"
)

type Collect struct {
	collect map[*wsClient.Client]struct{}
	mux     *sync.RWMutex
}

func New() *Collect {
	return &Collect{
		collect: map[*wsClient.Client]struct{}{},
		mux:     &sync.RWMutex{},
	}
}

func (r *Collect) Add(ws *wsClient.Client) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.collect[ws] = struct{}{}
}

func (r *Collect) Del(ws *wsClient.Client) {
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.collect, ws)
}

func (r *Collect) RandomGetUniqIds(num int) []string {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if num <= 0 {
		return nil
	}
	ret := make([]string, 0, num)
	for ws := range r.collect {
		if len(ret) > num {
			break
		}
		ret = append(ret, ws.UniqId)
	}
	return ret
}

func (r *Collect) RandomGetCustomerIds(num int) []string {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if num <= 0 {
		return nil
	}
	ret := make([]string, 0, num)
	for ws := range r.collect {
		if len(ret) > num {
			break
		}
		customerId := ws.GetCustomerId()
		if customerId == "" {
			break
		}
		ret = append(ret, ws.GetCustomerId())
	}
	return ret
}
