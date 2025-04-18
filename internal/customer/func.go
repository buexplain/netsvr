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

package customer

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"netsvr/configs"
	"netsvr/internal/metrics"
	"time"
)

// WriteMessage 发送数据
func WriteMessage(conn *websocket.Conn, data []byte) bool {
	if err := conn.SetWriteDeadline(time.Now().Add(configs.Config.Customer.SendDeadline)); err != nil {
		metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(data)))
		_ = conn.Close()
		return false
	}
	if err := conn.WriteMessage(configs.Config.Customer.SendMessageType, data); err == nil {
		metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(data)))
		return true
	}
	metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(data)))
	_ = conn.Close()
	return false
}
