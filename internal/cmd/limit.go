package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/limit"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// Limit 更新限流配置、获取网关中的限流配置的真实情况
func Limit(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol2.LimitReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.LimitReq failed")
		return
	}
	//更新限流配置
	if len(payload.Items) > 0 {
		for _, item := range payload.Items {
			limit.Manager.SetLimits(item.Num, item.WorkerIds)
		}
	}
	//返回网关中的限流配置的真实情况
	ret := &protocol2.LimitResp{}
	ret.CtxData = payload.CtxData
	ret.Items = limit.Manager.Count()
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
