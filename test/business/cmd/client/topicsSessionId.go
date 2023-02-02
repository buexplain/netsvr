package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	workerUtils "netsvr/test/business/utils"
	"strconv"
)

// TopicsSessionId 获取网关中的某几个主题的连接session id
func TopicsSessionId(currentSessionId uint32, _ string, _ string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(protocol.TopicsSessionId)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse protocol.TopicsSessionId error: %v", err)
		return
	}
	req := internalProtocol.TopicsSessionIdReq{ReCtx: &internalProtocol.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = int32(protocol.RouterTopicsSessionId)
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = workerUtils.StrToReadOnlyBytes(strconv.FormatInt(int64(currentSessionId), 10))
	req.Topics = payload.Topics
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicsSessionId
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
