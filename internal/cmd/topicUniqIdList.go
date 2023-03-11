package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	protocol2 "netsvr/pkg/protocol"
)

// TopicUniqIdList 获取网关中某个主题包含的uniqId
func TopicUniqIdList(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol2.TopicUniqIdListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicUniqIdListReq failed")
		return
	}
	if payload.Topic == "" {
		return
	}
	ret := &protocol2.TopicUniqIdListResp{}
	ret.CtxData = payload.CtxData
	ret.Topic = payload.Topic
	ret.UniqIds = topic.Topic.GetUniqIds(payload.Topic)
	route := &protocol2.Router{}
	route.Cmd = protocol2.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
