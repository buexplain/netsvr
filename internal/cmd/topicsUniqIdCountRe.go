package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicsUniqIdCountRe 获取网关中的某几个主题的连接数
func TopicsUniqIdCountRe(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.TopicsUniqIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.TopicsUniqIdCountReq error: %v", err)
		return
	}
	if len(payload.Topics) == 0 && payload.CountAll == false {
		return
	}
	ret := &protocol.TopicsUniqIdCountResp{}
	ret.ReCtx = payload.ReCtx
	ret.Items = map[string]int32{}
	if payload.CountAll == true {
		topic.Topic.CountAll(ret.Items)
	} else {
		topic.Topic.Count(payload.Topics, ret.Items)
	}
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_TopicsUniqIdCountRe
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
