package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol/reCtx"
	"netsvr/internal/protocol/toServer/reqTopicsConnCount"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/pkg/utils"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"strconv"
)

// TopicsConnCount 获取网关中的某几个主题的连接数
func TopicsConnCount(currentSessionId uint32, userStr string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(protocol.TopicsConnCount)
	if err := json.Unmarshal(utils.StrToBytes(param), target); err != nil {
		logging.Error("Parse protocol.TopicsConnCount request error: %v", err)
		return
	}
	req := reqTopicsConnCount.ReqTopicsConnCount{ReCtx: &reCtx.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = int32(protocol.RouterTopicsConnCount)
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = utils.StrToBytes(strconv.FormatInt(int64(currentSessionId), 10))
	req.Topics = target.Topics
	req.GetAll = target.GetAll
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_ReqTopicsConnCount
	toServerRoute.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
