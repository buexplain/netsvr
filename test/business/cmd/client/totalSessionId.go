package client

import (
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	workerUtils "netsvr/test/business/utils"
	"strconv"
)

// TotalSessionId 获取网关所有在线的session id
func TotalSessionId(currentSessionId uint32, _ string, _ string, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.TotalSessionIdReq{ReCtx: &internalProtocol.ReCtx{}}
	req.ReCtx.Cmd = int32(protocol.RouterTotalSessionId)
	req.ReCtx.Data = workerUtils.StrToReadOnlyBytes(strconv.FormatInt(int64(currentSessionId), 10))
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TotalSessionId
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
