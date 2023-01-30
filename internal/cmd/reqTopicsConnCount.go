package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/reqTopicsConnCount"
	"netsvr/internal/protocol/toWorker/respTopicsConnCount"
	toWorkerRouter "netsvr/internal/protocol/toWorker/router"
	workerManager "netsvr/internal/worker/manager"
)

// ReqTopicsConnCount 获取网关中的某几个主题的连接数
func ReqTopicsConnCount(param []byte, processor *workerManager.ConnProcessor) {
	req := reqTopicsConnCount.ReqTopicsConnCount{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal reqTopicsConnCount.ReqTopicsConnCount error: %v", err)
		return
	}
	items := map[string]int32{}
	session.Topics.CountConn(req.Topics, req.GetAll, items)
	data := &respTopicsConnCount.RespTopicsConnCount{}
	data.ReCtx = req.ReCtx
	data.Items = items
	route := &toWorkerRouter.Router{}
	route.Cmd = toWorkerRouter.Cmd_RespTopicsConnCount
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
