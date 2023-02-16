package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/utils"
)

// TopicListRe 处理worker发送过来的网关中的主题
func TopicListRe(param []byte, processor *connProcessor.ConnProcessor) {
	payload := &internalProtocol.TopicListResp{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TopicListResp error: %v", err)
		return
	}
	if payload.ReCtx.Cmd != int32(protocol.RouterTopicList) {
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = utils.BytesToReadOnlyString(payload.ReCtx.Data)
	msg := map[string]interface{}{"topics": payload.Topics}
	ret.Data = utils.NewResponse(protocol.RouterTopicList, map[string]interface{}{"code": 0, "message": "获取网关中的主题成功", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
