package cmd

import (
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/proto"
	netsvrProtocol "netsvr/pkg/protocol"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
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
func (broadcast) Request(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	target := new(BroadcastParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), target); err != nil {
		log.Logger.Error().Err(err).Msg("Parse BroadcastParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	ret := &netsvrProtocol.Broadcast{}
	msg := map[string]interface{}{"fromUser": fromUser, "message": target.Message}
	ret.Data = testUtils.NewResponse(protocol.RouterBroadcast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_Broadcast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
