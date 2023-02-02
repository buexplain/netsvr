package client

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	businessUtils "netsvr/test/business/utils"
	workerUtils "netsvr/test/business/utils"
)

// Subscribe 处理客户的订阅请求
func Subscribe(currentSessionId uint32, _ string, _ string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(protocol.Subscribe)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse protocol.Subscribe error: %v", err)
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	//提交订阅信息到网关
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Subscribe
	ret := &internalProtocol.Subscribe{}
	ret.SessionId = currentSessionId
	ret.Topics = payload.Topics
	ret.Data = businessUtils.NewResponse(protocol.RouterSubscribe, map[string]interface{}{"code": 0, "message": "订阅成功", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
