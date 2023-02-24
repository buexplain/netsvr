package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/userDb"
	"netsvr/test/protocol"
	businessUtils "netsvr/test/utils"
)

type broadcast struct{}

var Broadcast = broadcast{}

func (r broadcast) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterBroadcast, r.Request)
}

// BroadcastParam 客户端发送的广播信息
type BroadcastParam struct {
	Message string
}

// Request 向worker发起广播请求
func (broadcast) Request(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	target := new(BroadcastParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), target); err != nil {
		logging.Error("Parse BroadcastParam error: %v", err)
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	ret := &internalProtocol.Broadcast{}
	msg := map[string]interface{}{"fromUser": fromUser, "message": target.Message}
	ret.Data = businessUtils.NewResponse(protocol.RouterBroadcast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Broadcast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
