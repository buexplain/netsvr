package business

import (
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/internal/protocol/toServer/setUserLoginStatusOk"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

var setUserLoginStatusOkCh chan *setUserLoginStatusOk.SetUserLoginStatusOk

func init() {
	setUserLoginStatusOkCh = make(chan *setUserLoginStatusOk.SetUserLoginStatusOk, 1000)
	for i := 0; i < 10; i++ {
		go setUserLoginStatusOkChConsumer()
	}
}

func SetUserLoginStatusOk(data *setUserLoginStatusOk.SetUserLoginStatusOk) {
	setUserLoginStatusOkCh <- data
}

func setUserLoginStatusOkChConsumer() {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("%#v", r)
			go setUserLoginStatusOkChConsumer()
		}
	}()
	for v := range setUserLoginStatusOkCh {
		conn := manager.Manager.Get(v.SessionId)
		if conn == nil {
			continue
		}
		info, ok := conn.Session().(*session.Info)
		if !ok {
			continue
		}
		info.SetLoginStatusOk(v.UserInfo)
		if len(v.Data) > 0 {
			if err := conn.WriteMessage(websocket.TextMessage, v.Data); err != nil {
				logging.Debug("Error singleCast: %#v", err)
			}
		}
	}
}
