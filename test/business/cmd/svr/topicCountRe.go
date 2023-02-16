package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/utils"
)

// TopicCountRe 处理worker发送过来的网关中的主题数量
func TopicCountRe(param []byte, processor *connProcessor.ConnProcessor) {
	payload := &internalProtocol.TopicCountResp{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TopicCountResp error: %v", err)
		return
	}
	if payload.ReCtx.Cmd != int32(protocol.RouterTopicCount) {
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = utils.BytesToReadOnlyString(payload.ReCtx.Data)
	msg := map[string]interface{}{"count": payload.Count}
	ret.Data = utils.NewResponse(protocol.RouterTopicCount, map[string]interface{}{"code": 0, "message": "获取网关中的主题数量成功", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
