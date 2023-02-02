package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/utils"
)

// SessionInfo 处理worker发送过来的某个用户的session信息
func SessionInfo(param []byte, processor *connProcessor.ConnProcessor) {
	payload := &internalProtocol.SessionInfoResp{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.SessionInfoResp error: %v", err)
		return
	}
	if payload.ReCtx.Cmd != int32(protocol.RouterTopicList) {
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.SessionId = payload.SessionId
	msg := map[string]interface{}{"topics": payload.Topics}
	ret.Data = utils.NewResponse(protocol.RouterTopicList, map[string]interface{}{"code": 0, "message": "获取已订阅的主题成功", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
