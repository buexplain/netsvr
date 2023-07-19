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

package token

import (
	"encoding/hex"
	gTimer "github.com/antlabs/timer"
	"netsvr/configs"
	"netsvr/internal/timer"
	"sync"
)
import "crypto/rand"

type token struct {
	data map[string]struct{}
	mux  *sync.Mutex
}

var Token *token

func init() {
	Token = &token{
		data: make(map[string]struct{}, 1000),
		mux:  &sync.Mutex{},
	}
}

// Get 获取一个token
func (r *token) Get() string {
	b := make([]byte, 16)
	for {
		_, err := rand.Read(b)
		if err == nil && len(b) >= 16 {
			break
		}
	}
	s := hex.EncodeToString(b[0:16])
	r.mux.Lock()
	r.data[s] = struct{}{}
	r.mux.Unlock()
	//加个定时，到期删除，避免堆积
	var timeNode gTimer.TimeNoder
	timeNode = timer.Timer.AfterFunc(configs.Config.Customer.ConnOpenCustomUniqIdTokenExpire, func() {
		if timeNode != nil {
			timeNode.Stop()
		}
		r.mux.Lock()
		defer r.mux.Unlock()
		delete(r.data, s)
	})
	return s
}

// Exist 判断一个token是否存在，如果存在会删除它，并返回true，否则返回false
func (r *token) Exist(token string) bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	_, ok := r.data[token]
	if ok {
		//用后即删
		delete(r.data, token)
	}
	return ok
}
