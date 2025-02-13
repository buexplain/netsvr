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

package netsvrBusiness

type TaskSocketPoolManger struct {
	pools map[string]*TaskSocketPool
}

func NewTaskSocketPoolManger() *TaskSocketPoolManger {
	return &TaskSocketPoolManger{
		pools: make(map[string]*TaskSocketPool),
	}
}

func (t *TaskSocketPoolManger) Close() {
	pools := t.pools
	t.pools = make(map[string]*TaskSocketPool)
	for _, pool := range pools {
		pool.Close()
	}
}

func (t *TaskSocketPoolManger) AddSocket(taskSocketPool *TaskSocketPool) {
	t.pools[WorkerAddrConvertToHex(taskSocketPool.GetWorkerAddr())] = taskSocketPool
}

func (t *TaskSocketPoolManger) Count() int {
	return len(t.pools)
}

func (t *TaskSocketPoolManger) GetSockets() []*TaskSocket {
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

func (t *TaskSocketPoolManger) GetSocket(workerAddrAsHex string) *TaskSocket {
	pool, ok := t.pools[workerAddrAsHex]
	if !ok {
		return nil
	}
	return pool.Get()
}
