package business

import (
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/protocol/toServer/singleCast"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

var singleCastCh chan *singleCast.SingleCast

func init() {
	singleCastCh = make(chan *singleCast.SingleCast, 1000)
	for i := 0; i < 10; i++ {
		go singleCastChConsumer()
	}
}

func SingleCast(singleCast *singleCast.SingleCast) {
	singleCastCh <- singleCast
}

func singleCastChConsumer() {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("%#v", r)
			go singleCastChConsumer()
		}
	}()
	for v := range singleCastCh {
		conn := manager.Manager.Get(v.SessionId)
		if conn != nil {
			if err := conn.WriteMessage(websocket.TextMessage, v.Data); err != nil {
				logging.Debug("Error singleCast: %#v", err)
			}
		}
	}
}
