package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicUniqIdList 获取网关中某个主题包含的uniqId
func TopicUniqIdList(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.TopicUniqIdListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.TopicUniqIdListReq failed")
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
