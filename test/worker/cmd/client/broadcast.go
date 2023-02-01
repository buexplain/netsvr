package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol/toServer/broadcast"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/pkg/utils"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/userDb"
	workerUtils "netsvr/test/worker/utils"
)

// Broadcast 广播
func Broadcast(_ uint32, userStr string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(protocol.Broadcast)
	if err := json.Unmarshal(utils.StrToBytes(param), target); err != nil {
		logging.Error("Parse protocol.Broadcast request error: %v", err)
		return
	}
	currentUser := userDb.ParseNetSvrInfo(userStr)
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次广播操作
	toServerRoute.Cmd = toServerRouter.Cmd_Broadcast
	//构造网关需要的广播数据
	ret := &broadcast.Broadcast{}
	//构造客户端需要的数据
	msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
	ret.Data = workerUtils.NewResponse(protocol.RouterBroadcast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
