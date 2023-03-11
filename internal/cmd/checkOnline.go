package cmd

import (
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// CheckOnline 检查网关中是否包含某几个uniqId
func CheckOnline(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol2.CheckOnlineReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.CheckOnlineReq failed")
		return
	}
	ret := &protocol2.CheckOnlineResp{}
	ret.CtxData = payload.CtxData
	uniqIds := make([]string, 0, len(payload.UniqIds))
	for _, uniqId := range payload.UniqIds {
		if customerManager.Manager.Has(uniqId) {
			uniqIds = append(uniqIds, uniqId)
		}
	}
	ret.UniqIds = uniqIds
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
