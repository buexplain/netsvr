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

import "github.com/buexplain/netsvr-business-go/v3/contract"

// Manger 多网关 task 连接池的管理器，按网关地址维护各自的连接池
type Manger struct {
	pools map[string]*Pool
}

// NewManger 创建一个连接池管理器
func NewManger() *Manger {
	return &Manger{
		pools: make(map[string]*Pool),
	}
}

// Close 关闭所有连接池
func (t *Manger) Close() {
	pools := t.pools
	t.pools = make(map[string]*Pool)
	for _, pool := range pools {
		pool.Close()
	}
}

// AddSocket 按连接池的网关地址注册连接池
func (t *Manger) AddSocket(taskSocketPool *Pool) {
	t.pools[contract.AddrConvertToHex(taskSocketPool.GetAddr())] = taskSocketPool
}

// Count 获取网关（连接池）数量
func (t *Manger) Count() int {
	return len(t.pools)
}

// GetSockets 从每个连接池各取出一个连接；任一池取不到则整体失败，并归还已取出的连接
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

// GetSocket 按网关地址的 16 进制从对应连接池取出一个连接，池不存在时返回 nil
func (t *Manger) GetSocket(addrAsHex string) *TaskSocket {
	pool, ok := t.pools[addrAsHex]
	if !ok {
		return nil
	}
	return pool.Get()
}
