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

// TopicsConnCount 获取网关中的某几个主题的连接数
func TopicsConnCount(currentSessionId uint32, _ string, _ string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := protocol.TopicsConnCount{}
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse protocol.TopicsConnCount error: %v", err)
		return
	}
	req := internalProtocol.TopicsConnCountReq{ReCtx: &internalProtocol.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = int32(protocol.RouterTopicsConnCount)
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = workerUtils.StrToReadOnlyBytes(strconv.FormatInt(int64(currentSessionId), 10))
	req.Topics = payload.Topics
	req.GetAll = payload.GetAll
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicsConnCount
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
