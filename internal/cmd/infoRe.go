package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// InfoRe 获取连接的信息
func InfoRe(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.InfoReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.InfoReq error: %v", err)
		return
	}
	if payload.UniqId == "" {
		return
	}
	conn := customerManager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	session, ok := conn.Session().(*info.Info)
	if !ok {
		return
	}
	ret := &protocol.InfoResp{}
	ret.ReCtx = payload.ReCtx
	session.GetToProtocolInfoResp(ret)
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_InfoRe
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
