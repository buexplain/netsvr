package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/utils"
	"strconv"
)

// TopicsConnCount 处理worker发送过来的主题的连接数
func TopicsConnCount(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.TopicsConnCountResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TopicsConnCountResp error: %v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if payload.ReCtx.Cmd != int32(protocol.RouterTopicsConnCount) {
		return
	}
	//解析请求上下文中存储的session id
	targetSessionId, _ := strconv.ParseInt(string(payload.ReCtx.Data), 10, 64)
	if targetSessionId == 0 {
		return
	}
	//将结果单播给客户端
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.SessionId = uint32(targetSessionId)
	ret.Data = utils.NewResponse(protocol.RouterTopicsConnCount, map[string]interface{}{"code": 0, "message": "获取网关中主题的连接数成功", "data": payload.Items})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
