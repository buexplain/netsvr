package wsMetrics

import gMetrics "github.com/rcrowley/go-metrics"

type WsStatus struct {
	//在线人数
	Online gMetrics.Counter
	//发送的消息字节数
	Send gMetrics.Counter
	//接收的消息字节数
	Receive gMetrics.Counter
}

func New() *WsStatus {
	return &WsStatus{
		Online:  gMetrics.NewCounter(),
		Send:    gMetrics.NewCounter(),
		Receive: gMetrics.NewCounter(),
	}
}
