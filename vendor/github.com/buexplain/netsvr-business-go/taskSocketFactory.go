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

import "time"

type TaskSocketFactory struct {
	workerAddr     string
	receiveTimeout time.Duration
	sendTimeout    time.Duration
	connectTimeout time.Duration
}

func NewTaskSocketFactory(workerAddr string, receiveTimeout time.Duration, sendTimeout time.Duration, connectTimeout time.Duration) *TaskSocketFactory {
	return &TaskSocketFactory{
		workerAddr:     workerAddr,
		receiveTimeout: receiveTimeout,
		sendTimeout:    sendTimeout,
		connectTimeout: connectTimeout,
	}
}

func (t *TaskSocketFactory) Make(pool *TaskSocketPool) *TaskSocket {
	socket := NewTaskSocket(t.workerAddr, t.receiveTimeout, t.sendTimeout, t.connectTimeout, pool)
	if socket.Connect() {
		return socket
	}
	return nil
}

func (t *TaskSocketFactory) GetWorkerAddr() string {
	return t.workerAddr
}
