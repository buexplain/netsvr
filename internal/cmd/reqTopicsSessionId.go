package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/reqTopicsSessionId"
	"netsvr/internal/protocol/toWorker/respTopicsSessionId"
	toWorkerRouter "netsvr/internal/protocol/toWorker/router"
	workerManager "netsvr/internal/worker/manager"
)

// ReqTopicsSessionId 获取网关中的某几个主题的session id
func ReqTopicsSessionId(param []byte, processor *workerManager.ConnProcessor) {
	req := reqTopicsSessionId.ReqTopicsSessionId{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal reqTopicsSessionId.ReqTopicsSessionId error: %v", err)
		return
	}
	items := map[string]string{}
	session.Topics.Gets(req.Topics, items)
	data := &respTopicsSessionId.RespTopicsSessionId{}
	data.ReCtx = req.ReCtx
	data.Items = items
	route := &toWorkerRouter.Router{}
	route.Cmd = toWorkerRouter.Cmd_RespTopicsSessionId
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
