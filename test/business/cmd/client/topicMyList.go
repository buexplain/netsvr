package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
)

// TopicMyList 获取我已订阅的主题列表
func TopicMyList(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	//获取网关中的info信息，从中提取出主题列表
	//正常的业务逻辑应该是查询数据库获取用户订阅的主题
	req := &internalProtocol.InfoReq{ReCtx: &internalProtocol.ReCtx{}}
	req.ReCtx.Cmd = int32(protocol.RouterTopicMyList)
	req.UniqId = tf.UniqId
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_InfoRe
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
