package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicUniqIdList 获取网关中某个主题包含的uniqId
func TopicUniqIdList(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.TopicUniqIdListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.TopicUniqIdListReq error: %v", err)
		return
	}
	if payload.Topic == "" {
		return
	}
	ret := &protocol.TopicUniqIdListResp{}
	ret.CtxData = payload.CtxData
	ret.Topic = payload.Topic
	ret.UniqIds = topic.Topic.GetUniqIds(payload.Topic)
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
