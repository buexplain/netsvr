package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicList 获取网关中的主题
func TopicList(param []byte, processor *workerManager.ConnProcessor) {
	payload := &protocol.TopicListReq{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.TopicListReq error: %v", err)
		return
	}
	ret := &protocol.TopicListResp{}
	ret.CtxData = payload.CtxData
	ret.Topics = topic.Topic.Get()
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
