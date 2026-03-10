/**
* Copyright 2024 buexplain@qq.com
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

package taskSocket

import "github.com/buexplain/netsvr-business-go/v2/contract"

type Manger struct {
	pools map[string]*Pool
}

func NewManger() *Manger {
	return &Manger{
		pools: make(map[string]*Pool),
	}
}

func (t *Manger) Close() {
	pools := t.pools
	t.pools = make(map[string]*Pool)
	for _, pool := range pools {
		pool.Close()
	}
}

func (t *Manger) AddSocket(taskSocketPool *Pool) {
	t.pools[contract.AddrConvertToHex(taskSocketPool.GetAddr())] = taskSocketPool
}

func (t *Manger) Count() int {
	return len(t.pools)
}

func (t *Manger) GetSockets() []*TaskSocket {
	ret := make([]*TaskSocket, 0, len(t.pools))
	for _, pool := range t.pools {
		socket := pool.Get()
		if socket == nil {
			for _, s := range ret {
				s.Release()
			}
			ret = nil
			break
		}
		ret = append(ret, socket)
	}
	return ret
}

func (t *Manger) GetSocket(addrAsHex string) *TaskSocket {
	pool, ok := t.pools[addrAsHex]
	if !ok {
		return nil
	}
	return pool.Get()
}
