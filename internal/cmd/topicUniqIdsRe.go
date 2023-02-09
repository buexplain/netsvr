package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicUniqIdsRe 获取网关中的主题包含的uniqId
func TopicUniqIdsRe(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.TopicUniqIdsReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.TopicUniqIdsReq error: %v", err)
		return
	}
	if payload.Topic == "" {
		return
	}
	ret := &protocol.TopicUniqIdsResp{}
	ret.ReCtx = payload.ReCtx
	ret.Topic = payload.Topic
	ret.UniqIds = topic.Topic.Get(payload.Topic)
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_TopicUniqIdsRe
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
