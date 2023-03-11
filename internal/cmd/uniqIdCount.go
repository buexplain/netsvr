package cmd

import (
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// UniqIdCount 获取网关中uniqId的数量
func UniqIdCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol2.UniqIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.UniqIdCountReq failed")
		return
	}
	ret := &protocol2.UniqIdCountResp{}
	ret.CtxData = payload.CtxData
	ret.Count = int32(customerManager.Manager.Len())
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
