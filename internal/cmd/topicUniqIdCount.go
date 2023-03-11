package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// TopicUniqIdCount 获取网关中的主题包含的连接数
func TopicUniqIdCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol2.TopicUniqIdCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicUniqIdCountReq failed")
		return
	}
	if len(payload.Topics) == 0 && payload.CountAll == false {
		return
	}
	ret := &protocol2.TopicUniqIdCountResp{}
	ret.CtxData = payload.CtxData
	ret.Items = map[string]int32{}
	if payload.CountAll == true {
		topic.Topic.CountAll(ret.Items)
	} else {
		topic.Topic.Count(payload.Topics, ret.Items)
	}
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
