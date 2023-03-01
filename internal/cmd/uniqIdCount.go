package cmd

import (
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// UniqIdCount 获取网关中uniqId的数量
func UniqIdCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.UniqIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.UniqIdCountReq failed")
		return
	}
	ret := &protocol.UniqIdCountResp{}
	ret.CtxData = payload.CtxData
	ret.Count = int32(customerManager.Manager.Len())
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
