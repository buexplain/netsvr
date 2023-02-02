package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TotalSessionId 获取网关中全部的session id
func TotalSessionId(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.TotalSessionIdReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.TotalSessionIdReq error: %v", err)
		return
	}
	bitmap := session.Id.GetAllocated()
	data := &protocol.TotalSessionIdResp{}
	data.ReCtx = payload.ReCtx
	data.SessionIds, _ = bitmap.ToBase64()
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_TotalSessionId
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
