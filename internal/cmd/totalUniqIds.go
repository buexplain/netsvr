package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TotalUniqIds 获取网关中全部的uniqId
func TotalUniqIds(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.TotalUniqIdsReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.TotalUniqIdsReq error: %v", err)
		return
	}
	uniqIds := make([]string, 0, customerManager.Manager[0].Len()*len(customerManager.Manager))
	for _, c := range customerManager.Manager {
		c.GetUniqIds(&uniqIds)
	}
	ret := &protocol.TotalUniqIdsResp{}
	ret.CtxData = payload.CtxData
	ret.UniqIds = uniqIds
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
