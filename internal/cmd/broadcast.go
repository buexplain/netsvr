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

package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// Broadcast 广播
func Broadcast(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.Broadcast.Get()
	defer objPool.Broadcast.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.Broadcast failed")
		return
	}
	if len(payload.Data) == 0 {
		return
	}
	//取出所有的连接
	//加1024的目的是担心获取连接的过程中又有连接进来，导致slice的底层发生扩容，引起内存拷贝
	connections := objPool.ConnSlice.Get(customerManager.Manager.Len() + 1024)
	for _, c := range customerManager.Manager {
		c.GetConnections(connections)
	}
	//循环所有的连接，挨个发送出去
	for _, conn := range *connections {
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			_ = conn.Close()
		}
	}
	objPool.ConnSlice.Put(connections)
}
