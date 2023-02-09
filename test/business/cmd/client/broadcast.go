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

// Broadcast 广播
func Broadcast(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	target := new(protocol.Broadcast)
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), target); err != nil {
		logging.Error("Parse protocol.Broadcast request error: %v", err)
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId: %s", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	ret := &internalProtocol.Broadcast{}
	msg := map[string]interface{}{"fromUser": fromUser, "message": target.Message}
	ret.Data = workerUtils.NewResponse(protocol.RouterBroadcast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Broadcast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
