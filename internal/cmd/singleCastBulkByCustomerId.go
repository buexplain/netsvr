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
	"netsvr/internal/customer"
	"netsvr/internal/customer/binder"
	"netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// SingleCastBulkByCustomerId 批量单播
func SingleCastBulkByCustomerId(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.SingleCastBulkByCustomerId.Get()
	defer objPool.SingleCastBulkByCustomerId.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.SingleCastBulkByCustomerId failed")
		return
	}
	//当业务进程传递的customerIds的customerId数量只有一个，data的datum数量是一个以上时，网关必须将所有的datum都发送给这个customerId
	if len(payload.CustomerIds) == 1 && len(payload.Data) > 1 {
		uniqIds := binder.Binder.GetUniqIdsByCustomerId(payload.CustomerIds[0])
		for _, uniqId := range uniqIds {
			//根据uniqId获得对应的连接
			conn := manager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			//迭代所有数据
			var index int
			for index = range payload.Data {
				if len(payload.Data[index]) == 0 {
					continue
				}
				//将当前数据写入到连接中
				if !customer.WriteMessage(conn, payload.Data[index]) {
					//写入失败，直接跳出写数据循环，处理下一个连接，不必再处理剩余数据
					break
				}
			}
		}
		return
	}
	//当业务进程传递的customerIds的customerId数量与data的datum数量一致时，网关必须将同一下标的datum，发送给同一下标的customerId
	if len(payload.CustomerIds) > 0 && len(payload.CustomerIds) == len(payload.Data) {
		//迭代所有数据
		var conn *websocket.Conn
		var index int
		for index = range payload.Data {
			//判断数据是否有效
			if len(payload.Data[index]) == 0 {
				continue
			}
			//获得数据对应的index下标的customerId对应的uniqId
			uniqIds := binder.Binder.GetUniqIdsByCustomerId(payload.CustomerIds[index])
			for _, uniqId := range uniqIds {
				conn = manager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				//将数据写入到连接中
				customer.WriteMessage(conn, payload.Data[index])
			}
		}
	}
}
