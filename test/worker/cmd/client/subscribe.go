package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol/reCtx"
	"netsvr/internal/protocol/toServer/reqSessionInfo"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/subscribe"
	"netsvr/pkg/utils"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
)

// Subscribe 处理客户的订阅请求
func Subscribe(currentSessionId uint32, userStr string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(protocol.Subscribe)
	if err := json.Unmarshal(utils.StrToBytes(param), target); err != nil {
		logging.Error("Parse protocol.Subscribe request error: %v", err)
		return
	}
	if len(target.Topics) == 0 {
		return
	}
	//提交订阅信息到网关
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次订阅操作
	toServerRoute.Cmd = toServerRouter.Cmd_Subscribe
	//构造网关需要的订阅数据
	ret := &subscribe.Subscribe{}
	ret.SessionId = currentSessionId
	ret.Topics = target.Topics
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
	//查询该用户的订阅信息
	req := &reqSessionInfo.ReqSessionInfo{ReCtx: &reCtx.ReCtx{}}
	req.SessionId = currentSessionId
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = int32(protocol.RouterSubscribe)
	toServerRoute.Cmd = toServerRouter.Cmd_ReqSessionInfo
	toServerRoute.Data, _ = proto.Marshal(req)
	pt, _ = proto.Marshal(toServerRoute)
	processor.Send(pt)
}
