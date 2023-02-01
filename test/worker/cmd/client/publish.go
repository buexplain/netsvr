package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol/toServer/publish"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/pkg/utils"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/userDb"
	workerUtils "netsvr/test/worker/utils"
)

// Publish 处理客户的发布请求
func Publish(_ uint32, userStr string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(protocol.Publish)
	if err := json.Unmarshal(utils.StrToBytes(param), target); err != nil {
		logging.Error("Parse protocol.Publish request error: %v", err)
		return
	}
	currentUser := userDb.ParseNetSvrInfo(userStr)
	msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
	ret := &publish.Publish{}
	ret.Topic = target.Topic
	ret.Data = workerUtils.NewResponse(protocol.RouterPublish, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_Publish
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
