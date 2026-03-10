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

import (
	"github.com/buexplain/netsvr-business-go/v2/socket"
	"time"
)

type TaskSocket struct {
	*socket.Socket
	pool *Pool
}

func New(addr string, receiveTimeout time.Duration, sendTimeout time.Duration, connectTimeout time.Duration, pool *Pool) *TaskSocket {
	return &TaskSocket{
		Socket: socket.New(
			addr,
			receiveTimeout,
			sendTimeout,
			connectTimeout,
		),
		pool: pool,
	}
}

func (t *TaskSocket) Release() {
	t.pool.release(t)
}
