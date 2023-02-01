package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/singleCast"
	"netsvr/internal/protocol/toWorker/respTopicsSessionId"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/utils"
	"strconv"
)

// RespTopicsSessionId 处理网关发送的主题的连接session id
func RespTopicsSessionId(param []byte, processor *connProcessor.ConnProcessor) {
	resp := &respTopicsSessionId.RespTopicsSessionId{}
	if err := proto.Unmarshal(param, resp); err != nil {
		logging.Error("Proto unmarshal respTopicsSessionId.RespTopicsSessionId error:%v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if resp.ReCtx.Cmd != int32(protocol.RouterTopicsSessionId) {
		return
	}
	//解析请求上下文中存储的session id
	targetSessionId, _ := strconv.ParseInt(string(resp.ReCtx.Data), 10, 64)
	if targetSessionId == 0 {
		return
	}
	//将结果单播给客户端
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	ret := &singleCast.SingleCast{}
	ret.SessionId = uint32(targetSessionId)
	ret.Data = utils.NewResponse(protocol.RouterTopicsSessionId, map[string]interface{}{"code": 0, "message": "获取网关中的主题的连接session id成功", "data": resp.Items})
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
