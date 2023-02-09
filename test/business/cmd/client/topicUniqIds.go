package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	workerUtils "netsvr/test/business/utils"
)

// TopicUniqIds 获取网关中的某个主题包含的uniqId
func TopicUniqIds(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(protocol.TopicUniqIds)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse protocol.TopicUniqIds error: %v", err)
		return
	}
	req := internalProtocol.TopicUniqIdsReq{ReCtx: &internalProtocol.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = int32(protocol.RouterTopicUniqIds)
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = workerUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Topic = payload.Topic
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicUniqIdsRe
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
