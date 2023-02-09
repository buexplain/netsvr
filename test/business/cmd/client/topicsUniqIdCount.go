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

// TopicsUniqIdCount 获取网关中的某几个主题的连接数
func TopicsUniqIdCount(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := protocol.TopicsUniqIdCount{}
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse protocol.TopicsUniqIdCount error: %v", err)
		return
	}
	req := internalProtocol.TopicsUniqIdCountReq{ReCtx: &internalProtocol.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = int32(protocol.RouterTopicsUniqIdCount)
	//将uniqId存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = workerUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Topics = payload.Topics
	req.CountAll = payload.CountAll
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicsUniqIdCountRe
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
