package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// Info 获取连接的信息
func Info(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol2.InfoReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.InfoReq failed")
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
	ret := &protocol2.InfoResp{}
	ret.CtxData = payload.CtxData
	session.GetToProtocolInfoResp(ret)
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
