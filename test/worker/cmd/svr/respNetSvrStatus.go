package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/singleCast"
	"netsvr/internal/protocol/toWorker/respNetSvrStatus"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/utils"
	"strconv"
)

// RespNetSvrStatus 处理网关发送的网关状态信息
func RespNetSvrStatus(param []byte, processor *connProcessor.ConnProcessor) {
	resp := &respNetSvrStatus.RespNetSvrStatus{}
	if err := proto.Unmarshal(param, resp); err != nil {
		logging.Error("Proto unmarshal respNetSvrStatus.RespNetSvrStatus error:%v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if resp.ReCtx.Cmd != int32(protocol.RouterNetSvrStatus) {
		return
	}
	//解析请求上下文中存储的session id
	targetSessionId, _ := strconv.ParseInt(string(resp.ReCtx.Data), 10, 64)
	if targetSessionId == 0 {
		return
	}
	//将网关的信息单播给客户端
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	ret := &singleCast.SingleCast{}
	ret.SessionId = uint32(targetSessionId)
	msg := map[string]interface{}{
		"customerConnCount":     resp.CustomerConnCount,
		"topicCount":            resp.TopicCount,
		"catapultWaitSendCount": resp.CatapultWaitSendCount,
		"catapultConsumer":      resp.CatapultConsumer,
		"catapultChanCap":       resp.CatapultChanCap,
	}
	ret.Data = utils.NewResponse(protocol.RouterNetSvrStatus, map[string]interface{}{"code": 0, "message": "获取网关状态信息成功", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
