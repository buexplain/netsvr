package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/singleCast"
	"netsvr/pkg/utils"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/userDb"
	workerUtils "netsvr/test/worker/utils"
)

// SingleCast 单播
func SingleCast(currentSessionId uint32, userStr string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(protocol.SingleCast)
	if err := json.Unmarshal(utils.StrToBytes(param), target); err != nil {
		logging.Error("Parse protocol.SingleCast request error: %v", err)
		return
	}
	currentUser := userDb.ParseNetSvrInfo(userStr)
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次单播操作
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造网关需要的单播数据
	ret := &singleCast.SingleCast{}
	//查询目标用户的sessionId
	targetSessionId := userDb.Collect.GetSessionId(target.UserId)
	if targetSessionId == 0 {
		//目标用户不存在，返回信息给到发送者
		ret.SessionId = currentSessionId
		ret.Data = workerUtils.NewResponse(protocol.RouterSingleCast, map[string]interface{}{"code": 1, "message": "未找到目标用户"})
	} else {
		//目标用户存在，将信息转发给目标用户
		ret.SessionId = targetSessionId
		msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
		ret.Data = workerUtils.NewResponse(protocol.RouterSingleCast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	}
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
