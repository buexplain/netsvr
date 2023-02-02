package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
)

// TopicList 获取已订阅的主题列表
func TopicList(currentSessionId uint32, _ string, _ string, _ string, processor *connProcessor.ConnProcessor) {
	//查询该客户的session信息
	req := &internalProtocol.SessionInfoReq{ReCtx: &internalProtocol.ReCtx{}}
	req.ReCtx.Cmd = int32(protocol.RouterTopicList)
	req.SessionId = currentSessionId
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SessionInfo
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
