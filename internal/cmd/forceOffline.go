package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
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
	if conn != nil {
		_ = conn.Close()
	}
}
