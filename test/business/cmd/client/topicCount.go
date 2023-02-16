package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/utils"
)

// TopicCount 获取网关中的主题数量
func TopicCount(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.TopicCountReq{ReCtx: &internalProtocol.ReCtx{}}
	req.ReCtx.Cmd = int32(protocol.RouterTopicCount)
	req.ReCtx.Data = utils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicCountRe
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
