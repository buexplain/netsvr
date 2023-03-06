package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	"netsvr/internal/timer"
	workerManager "netsvr/internal/worker/manager"
	"time"
)

// ForceOfflineGuest 强制关闭某个空session值的连接
func ForceOfflineGuest(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.ForceOfflineGuest{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.ForceOfflineGuest failed")
		return
	}
	if payload.UniqId == "" {
		return
	}
	f := func() {
		conn := customerManager.Manager.Get(payload.UniqId)
		if conn == nil {
			return
		}
		//跳过有session值的连接
		if session, ok := conn.Session().(*info.Info); ok && session.GetSession() != "" {
			return
		}
		//判断是否转发数据
		if len(payload.Data) == 0 {
			//无须转发任何数据，直接关闭连接
			_ = conn.Close()
		} else {
			//写入数据，并在一定倒计时后关闭连接
			_ = conn.WriteMessage(websocket.TextMessage, payload.Data)
			timer.Timer.AfterFunc(time.Second*3, func() {
				defer func() {
					_ = recover()
				}()
				_ = conn.Close()
			})
		}
	}
	if payload.Delay <= 0 {
		f()
	} else {
		timer.Timer.AfterFunc(time.Second*time.Duration(payload.Delay), func() {
			f()
		})
	}
}
