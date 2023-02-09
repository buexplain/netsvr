package client

import (
	"encoding/json"
	"fmt"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/userDb"
	workerUtils "netsvr/test/business/utils"
)

// Publish 处理客户的发布请求
func Publish(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(protocol.Publish)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), target); err != nil {
		logging.Error("Parse protocol.Publish error: %v", err)
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	msg := map[string]interface{}{"fromUser": fromUser, "message": target.Message}
	ret := &internalProtocol.Publish{}
	ret.Topic = target.Topic
	ret.Data = workerUtils.NewResponse(protocol.RouterPublish, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Publish
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
