package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/customer/session"
	"netsvr/internal/metrics"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// NetSvrStatus 返回网关的状态
func NetSvrStatus(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.NetSvrStatusReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.NetSvrStatus error: %v", err)
		return
	}
	data := &protocol.NetSvrStatusResp{}
	data.ReCtx = payload.ReCtx
	data.CustomerConnCount = int32(session.Id.CountAllocated())
	data.TopicCount = int32(session.Topics.Count())
	data.CatapultWaitSendCount = int32(Catapult.CountWaitSend())
	data.CatapultConsumer = int32(configs.Config.CatapultConsumer)
	data.CatapultChanCap = int32(configs.Config.CatapultChanCap)
	data.MetricsCustomerConnOpen = metrics.Registry[metrics.ItemCustomerConnOpen].ToMap()
	data.MetricsCustomerConnClose = metrics.Registry[metrics.ItemCustomerConnClose].ToMap()
	data.MetricsCustomerHeartbeat = metrics.Registry[metrics.ItemCustomerHeartbeat].ToMap()
	data.MetricsCustomerTransfer = metrics.Registry[metrics.ItemCustomerTransfer].ToMap()
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_NetSvrStatus
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
