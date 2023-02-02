package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/userDb"
	workerUtils "netsvr/test/business/utils"
)

// Broadcast 广播
func Broadcast(_ uint32, userInfo string, _ string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(protocol.Broadcast)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), target); err != nil {
		logging.Error("Parse protocol.Broadcast request error: %v", err)
		return
	}
	currentUser := userDb.ParseNetSvrInfo(userInfo)
	//构建一个发给网关的路由
	router := &internalProtocol.Router{}
	//告诉网关要进行一次广播操作
	router.Cmd = internalProtocol.Cmd_Broadcast
	//构造网关需要的广播数据
	ret := &internalProtocol.Broadcast{}
	//构造客户端需要的数据
	msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
	ret.Data = workerUtils.NewResponse(protocol.RouterBroadcast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
