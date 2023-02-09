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

// Unsubscribe 处理客户的取消订阅请求
func Unsubscribe(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(protocol.Unsubscribe)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse protocol.Unsubscribe error: %v", err)
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Unsubscribe
	ret := &internalProtocol.Unsubscribe{}
	ret.UniqId = tf.UniqId
	ret.Topics = payload.Topics
	ret.Data = businessUtils.NewResponse(protocol.RouterSubscribe, map[string]interface{}{"code": 0, "message": "取消订阅成功", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
