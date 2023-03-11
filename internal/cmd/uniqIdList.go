package cmd

import (
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// UniqIdList 获取网关中全部的uniqId
func UniqIdList(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol2.UniqIdListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.UniqIdListReq failed")
		return
	}
	uniqIds := make([]string, 0, customerManager.Manager.Len())
	for _, c := range customerManager.Manager {
		c.GetUniqIds(&uniqIds)
	}
	ret := &protocol2.UniqIdListResp{}
	ret.CtxData = payload.CtxData
	ret.UniqIds = uniqIds
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
