package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/reqNetSvrStatus"
	"netsvr/internal/protocol/toWorker/respNetSvrStatus"
	toWorkerRouter "netsvr/internal/protocol/toWorker/router"
	workerManager "netsvr/internal/worker/manager"
)

// ReqNetSvrStatus 返回网关的状态
func ReqNetSvrStatus(param []byte, processor *workerManager.ConnProcessor) {
	req := reqNetSvrStatus.ReqNetSvrStatus{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal reqNetSvrStatus.ReqNetSvrStatus error: %v", err)
		return
	}
	data := &respNetSvrStatus.RespNetSvrStatus{}
	data.ReCtx = req.ReCtx
	data.CustomerConnCount = int32(session.Id.CountAllocated())
	data.TopicCount = int32(session.Topics.Count())
	data.CatapultWaitSendCount = int32(Catapult.CountWaitSend())
	data.CatapultConsumer = int32(configs.Config.CatapultConsumer)
	data.CatapultChanCap = int32(configs.Config.CatapultChanCap)
	route := &toWorkerRouter.Router{}
	route.Cmd = toWorkerRouter.Cmd_RespNetSvrStatus
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
