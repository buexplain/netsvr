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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/customer/binder"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/timer"
	workerManager "netsvr/internal/worker/manager"
	"time"
)

// ForceOfflineByCustomerId 将连接强制关闭
func ForceOfflineByCustomerId(param []byte, _ *workerManager.ConnProcessor) {
	payload := &netsvrProtocol.ForceOfflineByCustomerId{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.ForceOfflineByCustomerId failed")
		return
	}
	//获取customerId对应的uniqId列表
	customerIdUniqIds := binder.Binder.GetUniqIdsByCustomerIds(payload.CustomerIds)
	//判断是否转发数据
	if len(payload.Data) == 0 {
		//无须转发任何数据，直接关闭连接
		for _, uniqIds := range customerIdUniqIds {
			for _, uniqId := range uniqIds {
				conn := customerManager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				_ = conn.Close()
			}
		}
		return
	}
	//写入数据，并在一定倒计时后关闭连接
	for _, uniqIds := range customerIdUniqIds {
		for _, uniqId := range uniqIds {
			conn := customerManager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			if err := conn.WriteMessage(configs.Config.Customer.SendMessageType, payload.Data); err == nil {
				//倒计时的目的是确保数据发送成功
				timer.Timer.AfterFunc(time.Millisecond*100, func() {
					defer func() {
						_ = recover()
					}()
					_ = conn.Close()
				})
				metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(payload.Data)))
			} else {
				metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(payload.Data)))
				_ = conn.Close()
			}
		}
	}
}
