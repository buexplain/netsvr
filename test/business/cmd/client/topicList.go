package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
)

// TopicList 获取已订阅的主题列表
func TopicList(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	//获取网关中的info信息
	req := &internalProtocol.InfoReq{ReCtx: &internalProtocol.ReCtx{}}
	req.ReCtx.Cmd = int32(protocol.RouterTopicList)
	req.UniqId = tf.UniqId
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_InfoRe
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
