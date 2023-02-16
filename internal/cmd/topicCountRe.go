package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicCountRe 获取网关中的主题数量
func TopicCountRe(param []byte, processor *workerManager.ConnProcessor) {
	payload := &protocol.TopicCountReq{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.TopicCountReq error: %v", err)
		return
	}
	ret := &protocol.TopicCountResp{}
	ret.ReCtx = payload.ReCtx
	ret.Count = int32(topic.Topic.Len())
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_TopicCountRe
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
