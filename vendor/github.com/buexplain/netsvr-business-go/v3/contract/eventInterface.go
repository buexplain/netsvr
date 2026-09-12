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

package contract

import (
	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
)

// EventInterface 业务进程实现该接口以接收网关转发的事件。
// 三个方法的入参为协议生成类型，直接按字段读取即可。
type EventInterface interface {
	// OnOpen 连接打开事件
	OnOpen(connOpen *netsvrProtocol.ConnOpen)
	// OnMessage 连接收到消息事件
	OnMessage(transfer *netsvrProtocol.Transfer)
	// OnClose 连接关闭事件
	OnClose(connClose *netsvrProtocol.ConnClose)
}
