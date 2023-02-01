package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/singleCast"
	"netsvr/internal/protocol/toWorker/respSessionInfo"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/utils"
)

// RespSessionInfo 处理网关发送的某个用户的session信息
func RespSessionInfo(param []byte, processor *connProcessor.ConnProcessor) {
	resp := &respSessionInfo.RespSessionInfo{}
	if err := proto.Unmarshal(param, resp); err != nil {
		logging.Error("Proto unmarshal respSessionInfo.RespSessionInfo error:%v", err)
		return
	}
	//根据自定义的cmd判断到底谁发起的请求网关session info信息
	if resp.ReCtx.Cmd == int32(protocol.RouterSubscribe) {
		respSessionInfoBySubscribe(resp, processor)
	} else if resp.ReCtx.Cmd == int32(protocol.RouterUnsubscribe) {
		respSessionInfoByUnsubscribe(resp, processor)
	}
}

// 用户订阅后拉取的用户信息
func respSessionInfoBySubscribe(resp *respSessionInfo.RespSessionInfo, processor *connProcessor.ConnProcessor) {
	if len(resp.Topics) == 0 {
		return
	}
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次单播操作
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造网关需要的单播数据
	ret := &singleCast.SingleCast{}
	ret.SessionId = resp.SessionId
	msg := map[string]interface{}{"topics": resp.Topics}
	ret.Data = utils.NewResponse(protocol.RouterSubscribe, map[string]interface{}{"code": 0, "message": "订阅成功", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}

// 用户取消订阅后拉取的用户信息
func respSessionInfoByUnsubscribe(resp *respSessionInfo.RespSessionInfo, processor *connProcessor.ConnProcessor) {
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次单播操作
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造网关需要的单播数据
	ret := &singleCast.SingleCast{}
	ret.SessionId = resp.SessionId
	msg := map[string]interface{}{"topics": resp.Topics}
	ret.Data = utils.NewResponse(protocol.RouterUnsubscribe, map[string]interface{}{"code": 0, "message": "取消订阅成功", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
