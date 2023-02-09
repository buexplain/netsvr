package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	workerUtils "netsvr/test/business/utils"
)

// TotalUniqIds 获取网关所有的uniqId
func TotalUniqIds(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.TotalUniqIdsReq{ReCtx: &internalProtocol.ReCtx{}}
	req.ReCtx.Cmd = int32(protocol.RouterTotalUniqIds)
	req.ReCtx.Data = workerUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TotalUniqIdsRe
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
