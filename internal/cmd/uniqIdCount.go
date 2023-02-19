package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// UniqIdCount 获取网关中uniqId的数量
func UniqIdCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.UniqIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.UniqIdCountReq error: %v", err)
		return
	}
	ret := &protocol.UniqIdCountResp{}
	ret.CtxData = payload.CtxData
	for _, c := range customerManager.Manager {
		ret.Count += int32(c.Len())
	}
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
