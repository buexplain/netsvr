package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// Metrics 返回网关统计的服务状态
func Metrics(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol2.MetricsReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.MetricsReq failed")
		return
	}
	ret := &protocol2.MetricsResp{}
	ret.CtxData = payload.CtxData
	ret.Items = map[int32]*protocol2.MetricsStatusResp{}
	for _, v := range metrics.Registry {
		if tmp := v.ToStatusResp(); tmp != nil {
			ret.Items[int32(v.Item)] = tmp
		}
	}
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
