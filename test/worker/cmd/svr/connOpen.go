package svr

import (
	"fmt"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/singleCast"
	"netsvr/internal/protocol/toWorker/connOpen"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/utils"
)

// ConnOpen 客户端打开连接
func ConnOpen(param []byte, processor *connProcessor.ConnProcessor) {
	req := connOpen.ConnOpen{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal connOpen.ConnOpen error:%v", err)
		return
	}
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造单播数据
	ret := &singleCast.SingleCast{}
	ret.SessionId = req.SessionId
	ret.Data = utils.NewResponse(protocol.RouterRespConnOpen, map[string]interface{}{"code": 0, "message": fmt.Sprintf("连接网关成功，sessionId: %d", req.SessionId)})
	//将业务数据放到路由上
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
