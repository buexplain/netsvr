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

// CheckOnlineForUniqId 检查某几个连接是否在线
func CheckOnlineForUniqId(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := protocol.CheckOnlineForUniqId{}
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse protocol.CheckOnlineForUniqId request error: %v", err)
		return
	}
	req := &internalProtocol.CheckOnlineReq{ReCtx: &internalProtocol.ReCtx{}}
	req.ReCtx.Cmd = int32(protocol.RouterCheckOnlineForUniqId)
	req.ReCtx.Data = workerUtils.StrToReadOnlyBytes(tf.UniqId)
	req.UniqIds = payload.UniqIds
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_CheckOnlineRe
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
