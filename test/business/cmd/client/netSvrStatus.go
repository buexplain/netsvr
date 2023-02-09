package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	workerUtils "netsvr/test/business/utils"
)

// NetSvrStatus 获取网关的信息
func NetSvrStatus(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.NetSvrStatusReq{ReCtx: &internalProtocol.ReCtx{}}
	req.ReCtx.Cmd = int32(protocol.RouterNetSvrStatus)
	req.ReCtx.Data = workerUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_NetSvrStatusRe
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
