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
	"sync"
)

type packet struct {
	header []byte
	body   []byte
}

type packetPool struct {
	pool *sync.Pool
}

var packetObjPool *packetPool

func init() {
	packetObjPool = &packetPool{
		pool: &sync.Pool{
			New: func() any {
				return &packet{}
			},
		},
	}
}

func (r *packetPool) Get() *packet {
	return r.pool.Get().(*packet)
}

func (r *packetPool) Put(packet *packet) {
	packet.header = nil
	packet.body = nil
	r.pool.Put(packet)
}
