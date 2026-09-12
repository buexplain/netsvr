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
	"netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/objPool"

	"google.golang.org/protobuf/proto"
)

// singleCastBulk 批量单播
func singleCastBulk(param []byte) {
	payload := objPool.SingleCastBulk.Get()
	defer objPool.SingleCastBulk.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.singleCastBulk failed")
		return
	}
	//迭代每一项：本项内每一条数据按顺序发给本项内每一个目标
	for _, item := range payload.Items {
		for _, data := range item.GetData() {
			//判断数据是否有效
			if len(data) == 0 {
				continue
			}
			for _, uniqId := range item.GetUniqIds() {
				//根据uniqId获得对应的连接
				conn := manager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				//将数据写入到连接中
				if !customer.WriteMessage(conn, configs.Config.Customer.SendMessageType, data) {
					//写入失败，直接退出，不必再处理剩余数据
					return
				}
			}
		}
	}
}
