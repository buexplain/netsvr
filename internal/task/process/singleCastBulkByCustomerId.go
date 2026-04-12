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
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/customer"
	"netsvr/internal/customer/binder"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
)

// singleCastBulkByCustomerId 批量单播
func singleCastBulkByCustomerId(param []byte) {
	payload := objPool.SingleCastBulkByCustomerId.Get()
	defer objPool.SingleCastBulkByCustomerId.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.singleCastBulkByCustomerId failed")
		return
	}
	//当业务进程传递的customerIds的customerId数量只有一个，data的datum数量是一个以上时，网关必须将所有的datum都发送给这个customerId
	if len(payload.CustomerIds) == 1 && len(payload.Data) > 1 {
		connList := binder.Binder.GetConnListByCustomerId(payload.CustomerIds[0])
		for _, data := range payload.Data {
			if len(data) == 0 {
				continue
			}
			msg := customer.NewMessage(configs.Config.Customer.SendMessageType, data)
			for _, conn := range connList {
				msg.WriteTo(conn)
			}
		}
		return
	}
	//当业务进程传递的customerIds的customerId数量与data的datum数量一致时，网关必须将同一下标的datum，发送给同一下标的customerId
	if len(payload.CustomerIds) > 0 && len(payload.CustomerIds) == len(payload.Data) {
		//迭代所有数据
		for index := range payload.Data {
			//判断数据是否有效
			if len(payload.Data[index]) == 0 {
				continue
			}
			//获得数据对应的index下标的customerId对应的uniqId
			connList := binder.Binder.GetConnListByCustomerId(payload.CustomerIds[index])
			msg := customer.NewMessage(configs.Config.Customer.SendMessageType, payload.Data[index])
			for _, conn := range connList {
				//将数据写入到连接中
				msg.WriteTo(conn)
			}
		}
		return
	}
	if len(payload.CustomerIds) > 0 && len(payload.CustomerIds) != len(payload.Data) {
		log.Logger.Warn().
			Int("customerIdsLen", len(payload.CustomerIds)).
			Int("dataLen", len(payload.Data)).
			Msg("singleCastBulkByCustomerId: length mismatch, message dropped")
	}
}
