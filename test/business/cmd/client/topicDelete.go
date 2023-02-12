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

// TopicDelete 处理客户的删除主题请求
func TopicDelete(_ *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(protocol.TopicDelete)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse protocol.TopicDelete error: %v", err)
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicDelete
	ret := &internalProtocol.TopicDelete{}
	ret.Topics = payload.Topics
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicDelete, map[string]interface{}{"code": 0, "message": "删除主题成功", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
