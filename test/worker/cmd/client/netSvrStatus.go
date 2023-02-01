package client

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol/reCtx"
	"netsvr/internal/protocol/toServer/reqNetSvrStatus"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/pkg/utils"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"strconv"
)

// NetSvrStatus 获取网关的信息
func NetSvrStatus(currentSessionId uint32, _ string, _ string, processor *connProcessor.ConnProcessor) {
	req := &reqNetSvrStatus.ReqNetSvrStatus{ReCtx: &reCtx.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = int32(protocol.RouterNetSvrStatus)
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = utils.StrToBytes(strconv.FormatInt(int64(currentSessionId), 10))
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_ReqNetSvrStatus
	toServerRoute.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
