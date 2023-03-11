package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// TopicList 获取网关中的主题
func TopicList(param []byte, processor *workerManager.ConnProcessor) {
	payload := &protocol2.TopicListReq{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicListReq failed")
		return
	}
	ret := &protocol2.TopicListResp{}
	ret.CtxData = payload.CtxData
	ret.Topics = topic.Topic.Get()
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
