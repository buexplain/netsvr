package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// SessionInfo 根据session id获取网关中的用户信息
func SessionInfo(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.SessionInfoReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.SessionInfoReq error: %v", err)
		return
	}
	data := &protocol.SessionInfoResp{}
	data.SessionId = payload.SessionId
	data.ReCtx = payload.ReCtx
	wsConn := customerManager.Manager.Get(payload.SessionId)
	if wsConn != nil {
		if info, ok := wsConn.Session().(*session.Info); ok {
			info.GetToSessionInfoObj(data)
		}
	}
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_SessionInfo
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
