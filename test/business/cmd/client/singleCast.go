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

// SingleCast 单播
func SingleCast(currentSessionId uint32, userInfo string, _ string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(protocol.SingleCast)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), target); err != nil {
		logging.Error("Parse protocol.SingleCast request error: %v", err)
		return
	}
	currentUser := userDb.ParseNetSvrInfo(userInfo)
	//构建一个发给网关的路由
	router := &internalProtocol.Router{}
	//告诉网关要进行一次单播操作
	router.Cmd = internalProtocol.Cmd_SingleCast
	//构造网关需要的单播数据
	ret := &internalProtocol.SingleCast{}
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
	router.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
