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
