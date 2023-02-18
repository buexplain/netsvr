package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/catapult"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/metrics"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// NetSvrStatus 返回网关的状态
func NetSvrStatus(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.NetSvrStatusReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.NetSvrStatusReq error: %v", err)
		return
	}
	ret := &protocol.NetSvrStatusResp{}
	ret.CtxData = payload.CtxData
	for _, c := range customerManager.Manager {
		ret.CustomerConnCount += int32(c.Len())
	}
	ret.Catapult = &protocol.CatapultResp{}
	ret.Catapult.ChanLen = int32(catapult.Catapult.Len())
	ret.Catapult.Consumer = int32(configs.Config.CatapultConsumer)
	ret.Catapult.ChanCap = int32(configs.Config.CatapultChanCap)
	ret.Metrics = map[string]*protocol.MetricsStatusResp{}
	for _, v := range metrics.Registry {
		if tmp := v.ToStatusResp(); tmp != nil {
			ret.Metrics[v.Name] = tmp
		}
	}
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
