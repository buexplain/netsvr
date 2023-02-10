package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// CheckOnlineRe 检查网关中是否包含某几个uniqId
func CheckOnlineRe(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.CheckOnlineReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.CheckOnlineReq error: %v", err)
		return
	}
	ret := &protocol.CheckOnlineResp{}
	ret.ReCtx = payload.ReCtx
	uniqIds := make([]string, 0, len(payload.UniqIds))
	for _, uniqId := range payload.UniqIds {
		if customerManager.Manager.Has(uniqId) {
			uniqIds = append(uniqIds, uniqId)
		}
	}
	ret.UniqIds = uniqIds
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_CheckOnlineRe
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
