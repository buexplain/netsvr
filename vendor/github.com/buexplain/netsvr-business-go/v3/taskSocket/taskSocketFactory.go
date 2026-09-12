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

import "time"

// Factory 创建 TaskSocket 的工厂，绑定目标网关地址与各阶段超时
type Factory struct {
	addr           string
	receiveTimeout time.Duration
	sendTimeout    time.Duration
	connectTimeout time.Duration
}

// NewFactory 创建一个 task 连接工厂
func NewFactory(addr string, receiveTimeout time.Duration, sendTimeout time.Duration, connectTimeout time.Duration) *Factory {
	return &Factory{
		addr:           addr,
		receiveTimeout: receiveTimeout,
		sendTimeout:    sendTimeout,
		connectTimeout: connectTimeout,
	}
}

// Make 创建一个 TaskSocket 并立即建立连接，连接失败返回 nil
func (t *Factory) Make(pool *Pool) *TaskSocket {
	socket := New(t.addr, t.receiveTimeout, t.sendTimeout, t.connectTimeout, pool)
	if socket.Connect() {
		return socket
	}
	return nil
}

// GetAddr 获取工厂对应的网关地址
func (t *Factory) GetAddr() string {
	return t.addr
}
