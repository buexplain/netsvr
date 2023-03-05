package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/limit"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// Limit 更新限流配置、获取网关中的限流配置的真实情况
func Limit(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.LimitReq{}
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
	ret := &protocol.LimitResp{}
	ret.CtxData = payload.CtxData
	ret.Items = limit.Manager.Count()
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
