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
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// SingleCast 单播
func SingleCast(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.SingleCast.Get()
	defer objPool.SingleCast.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.SingleCast failed")
		return
	}
	if payload.UniqId == "" || len(payload.Data) == 0 {
		return
	}
	conn := manager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	if err := conn.WriteMessage(configs.Config.Customer.SendMessageType, payload.Data); err == nil {
		metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(payload.Data)))
	} else {
		metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(payload.Data)))
		_ = conn.Close()
	}
}
