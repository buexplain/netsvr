package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	workerUtils "netsvr/test/business/utils"
	"strconv"
)

// NetSvrStatus 获取网关的信息
func NetSvrStatus(currentSessionId uint32, _ string, _ string, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.NetSvrStatusReq{ReCtx: &internalProtocol.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = int32(protocol.RouterNetSvrStatus)
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = workerUtils.StrToReadOnlyBytes(strconv.FormatInt(int64(currentSessionId), 10))
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_NetSvrStatus
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
