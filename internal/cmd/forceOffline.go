package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
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
	if payload.SessionId == 0 {
		return
	}
	conn := customerManager.Manager.Get(payload.SessionId)
	if conn == nil {
		return
	}
	//没有传递userId，则不做比较，直接关闭连接
	if payload.UserId == "" {
		_ = conn.Close()
		return
	}
	//比较userId后再做关闭连接的操作
	info, ok := conn.Session().(*session.Info)
	if !ok {
		_ = conn.Close()
		return
	}
	//连接id对应的客户已经被顶了，则不做关闭连接的操作
	userId := info.GetUserId()
	if userId != "" && userId != payload.UserId {
		return
	}
	_ = conn.Close()
}
