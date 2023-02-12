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

// TopicUnsubscribe 处理客户的取消订阅请求
func TopicUnsubscribe(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(protocol.TopicUnsubscribe)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse protocol.UnsubscribeTopics error: %v", err)
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicUnsubscribe
	ret := &internalProtocol.TopicUnsubscribe{}
	ret.UniqId = tf.UniqId
	ret.Topics = payload.Topics
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicSubscribe, map[string]interface{}{"code": 0, "message": "取消订阅成功", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
