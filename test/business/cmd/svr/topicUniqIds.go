package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/utils"
)

// TopicUniqIds 处理worker发送过来的主题的连接session id
func TopicUniqIds(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.TopicUniqIdsResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TopicUniqIdsResp error: %v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if payload.ReCtx.Cmd != int32(protocol.RouterTopicUniqIds) {
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := utils.BytesToReadOnlyString(payload.ReCtx.Data)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = utils.NewResponse(protocol.RouterTopicUniqIds, map[string]interface{}{"code": 0, "message": "获取网关中的主题的uniqId成功", "data": payload.UniqIds})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
