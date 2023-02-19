package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicUniqIdCount 获取网关中的主题包含的连接数
func TopicUniqIdCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.TopicUniqIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.TopicUniqIdCountReq error: %v", err)
		return
	}
	if len(payload.Topics) == 0 && payload.CountAll == false {
		return
	}
	ret := &protocol.TopicUniqIdCountResp{}
	ret.CtxData = payload.CtxData
	ret.Items = map[string]int32{}
	if payload.CountAll == true {
		topic.Topic.CountAll(ret.Items)
	} else {
		topic.Topic.Count(payload.Topics, ret.Items)
	}
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
