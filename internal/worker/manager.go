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

package worker

import (
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"math/rand/v2"
	"netsvr/configs"
	"netsvr/internal/log"
	"sync"
)

// 数组大小基于协议中最大的 Event 枚举值
const managerLen = netsvrProtocol.Event_OnMessage + 1

type manager [managerLen]*collect

func (r manager) Get(event netsvrProtocol.Event) *Conn {
	if configs.Config.Worker.ListenAddress == "" {
		return nil
	}
	return r[event].Get()
}

func (r manager) Set(conn *Conn) {
	events := conn.GetEvents()
	for _, v := range netsvrProtocol.Event_value {
		if netsvrProtocol.Event(v) == netsvrProtocol.Event_Placeholder {
			continue
		}
		if events&v == v {
			r[netsvrProtocol.Event(v)].Set(conn)
		}
	}
}

func (r manager) Del(connId string) bool {
	ret := false
	for _, v := range netsvrProtocol.Event_value {
		if netsvrProtocol.Event(v) == netsvrProtocol.Event_Placeholder {
			continue
		}
		if r[netsvrProtocol.Event(v)].Del(connId) {
			ret = true
		}
	}
	return ret
}

// Manager 管理所有的business连接
var Manager manager

func init() {
	if configs.Config.Worker.ListenAddress == "" {
		return
	}
	// 验证协议中的 Event 枚举值是否超出数组范围
	maxUsedEvent := 0
	for _, v := range netsvrProtocol.Event_value {
		maxUsedEvent = max(maxUsedEvent, int(v))
	}
	if maxUsedEvent >= int(managerLen) {
		log.Logger.Error().Msgf("Event enum value %d exceeds manager array size %d",
			maxUsedEvent, managerLen)
		panic("too many netsvrProtocol.Event")
	}
	Manager = manager{}
	for _, v := range netsvrProtocol.Event_value {
		if netsvrProtocol.Event(v) == netsvrProtocol.Event_Placeholder {
			continue
		}
		Manager[netsvrProtocol.Event(v)] = &collect{conn: []*Conn{}, index: rand.Uint32(), mux: sync.RWMutex{}}
	}
}
