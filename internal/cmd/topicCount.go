package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// TopicCount 获取网关中的主题数量
func TopicCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := &protocol2.TopicCountReq{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicCountReq failed")
		return
	}
	ret := &protocol2.TopicCountResp{}
	ret.CtxData = payload.CtxData
	ret.Count = int32(topic.Topic.Len())
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
