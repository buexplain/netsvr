package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/metrics"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// Metrics 返回网关统计的服务状态
func Metrics(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.MetricsReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.MetricsReq error: %v", err)
		return
	}
	ret := &protocol.MetricsResp{}
	ret.CtxData = payload.CtxData
	ret.Items = map[int32]*protocol.MetricsStatusResp{}
	for _, v := range metrics.Registry {
		if tmp := v.ToStatusResp(); tmp != nil {
			ret.Items[int32(v.Item)] = tmp
		}
	}
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
