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

package process

import (
	"netsvr/configs"
	"netsvr/internal/customer"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/objPool"

	"google.golang.org/protobuf/proto"
)

// broadcastBulk 批量广播
func broadcastBulk(param []byte) {
	payload := objPool.BroadcastBulk.Get()
	defer objPool.BroadcastBulk.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.broadcastBulk failed")
		return
	}
	if len(payload.Data) == 0 {
		return
	}
	//取出所有的连接
	connections := customerManager.Manager.GetConnections(objPool.ConnSlice)
	if connections == nil {
		return
	}
	defer objPool.ConnSlice.Put(connections)
	connectionsAlias := *connections //搞个别名，避免循环中解指针，提高性能
	//按顺序依次将每一条数据广播给全部连接
	for _, data := range payload.Data {
		if len(data) == 0 {
			continue
		}
		msg := customer.FrameObjPool.Get(configs.Config.Customer.SendMessageType, data)
		for _, conn := range connectionsAlias {
			msg.WriteTo(conn)
		}
		customer.FrameObjPool.Put(msg)
	}
}
