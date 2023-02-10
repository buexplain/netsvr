package cmd

import (
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/heartbeat"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
	"time"
)

// ForceOffline 将连接强制关闭
func ForceOffline(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.ForceOffline{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.ForceOffline error: %v", err)
		return
	}
	if payload.UniqId == "" {
		return
	}
	conn := customerManager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	//是否阻止worker把连接关闭事件转发到business，如果是，则立马移除掉连接的所有数据，这样在onClose的时候就不会转发数据到business
	if payload.PreventConnCloseCmdTransfer {
		//从连接管理器中删除
		customerManager.Manager.Del(payload.UniqId)
		//删除订阅关系、删除uniqId、关闭心跳
		if session, ok := conn.Session().(*info.Info); ok {
			topics, _, _ := session.Clear()
			topic.Topic.Del(topics, payload.UniqId)
		}
	}
	//判断是否转发数据
	if len(payload.Data) == 0 {
		//无须转发任何数据，直接关闭连接
		_ = conn.Close()
	} else {
		//写入数据，并在一定倒计时后关闭连接
		_ = conn.WriteMessage(websocket.TextMessage, payload.Data)
		heartbeat.Timer.AfterFunc(time.Second*3, func() {
			_ = conn.Close()
		})
	}
}
