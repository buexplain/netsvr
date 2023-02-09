package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TotalUniqIdsRe 获取网关中全部的uniqId
func TotalUniqIdsRe(param []byte, processor *workerManager.ConnProcessor) {
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
	ret.ReCtx = payload.ReCtx
	ret.UniqIds = uniqIds
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_TotalUniqIdsRe
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
