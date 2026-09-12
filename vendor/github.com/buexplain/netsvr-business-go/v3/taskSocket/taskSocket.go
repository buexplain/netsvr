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
	"github.com/buexplain/netsvr-business-go/v3/socket"
	"time"
)

// TaskSocket 与网关 task 服务的一条连接，使用完必须调用 Release 归还给连接池
type TaskSocket struct {
	*socket.Socket
	pool *Pool
}

// New 创建一个 task 连接对象，此时未建立连接，需自行调用 Connect
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

// Release 将连接归还给所属连接池；连接不可用时由池回收该名额
func (t *TaskSocket) Release() {
	t.pool.release(t)
}
