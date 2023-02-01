package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/singleCast"
	"netsvr/internal/protocol/toWorker/respTotalSessionId"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/utils"
	"strconv"
)

// RespTotalSessionId 处理网关发送的所有在线的session id
func RespTotalSessionId(param []byte, processor *connProcessor.ConnProcessor) {
	resp := &respTotalSessionId.RespTotalSessionId{}
	if err := proto.Unmarshal(param, resp); err != nil {
		logging.Error("Proto unmarshal respTotalSessionId.RespTotalSessionId error:%v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if resp.ReCtx.Cmd != int32(protocol.RouterTotalSessionId) {
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
	msg := map[string]interface{}{
		"totalSessionId": resp.Bitmap,
	}
	ret.Data = utils.NewResponse(protocol.RouterTotalSessionId, map[string]interface{}{"code": 0, "message": "获取网关所有在线的session id成功", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
