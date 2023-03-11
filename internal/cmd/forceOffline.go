package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/timer"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/protocol"
	"time"
)

// ForceOffline 将连接强制关闭
func ForceOffline(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.ForceOffline{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.ForceOffline failed")
		return
	}
	if len(payload.UniqIds) == 0 {
		return
	}
	//判断是否转发数据
	if len(payload.Data) == 0 {
		//无须转发任何数据，直接关闭连接
		for _, uniqId := range payload.UniqIds {
			conn := customerManager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			_ = conn.Close()
		}
		return
	}
	//写入数据，并在一定倒计时后关闭连接
	for _, uniqId := range payload.UniqIds {
		conn := customerManager.Manager.Get(uniqId)
		if conn == nil {
			continue
		}
		_ = conn.WriteMessage(websocket.TextMessage, payload.Data)
		timer.Timer.AfterFunc(time.Second*3, func() {
			defer func() {
				_ = recover()
			}()
			_ = conn.Close()
		})
	}
}
