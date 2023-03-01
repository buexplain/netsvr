package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicCount 获取网关中的主题数量
func TopicCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := &protocol.TopicCountReq{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicCountReq failed")
		return
	}
	ret := &protocol.TopicCountResp{}
	ret.CtxData = payload.CtxData
	ret.Count = int32(topic.Topic.Len())
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
