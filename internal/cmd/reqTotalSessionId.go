package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/reqTotalSessionId"
	"netsvr/internal/protocol/toWorker/respTotalSessionId"
	toWorkerRouter "netsvr/internal/protocol/toWorker/router"
	workerManager "netsvr/internal/worker/manager"
)

// ReqTotalSessionId 获取网关中全部的session id
func ReqTotalSessionId(param []byte, processor *workerManager.ConnProcessor) {
	req := reqTotalSessionId.ReqTotalSessionId{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal reqTotalSessionId.ReqTotalSessionId error: %v", err)
		return
	}
	bitmap := session.Id.GetAllocated()
	data := &respTotalSessionId.RespTotalSessionId{}
	data.ReCtx = req.ReCtx
	data.Bitmap, _ = bitmap.ToBase64()
	route := &toWorkerRouter.Router{}
	route.Cmd = toWorkerRouter.Cmd_RespTotalSessionId
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
