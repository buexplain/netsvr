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
	"strconv"
)

// ForceOffline 某个session id的连接强制关闭
func ForceOffline(_ uint32, _ string, _ string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := protocol.ForceOffline{}
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse protocol.ForceOffline request error: %v", err)
		return
	}
	user := userDb.Collect.GetUserById(payload.UserId)
	if user == nil || user.SessionId == 0 {
		return
	}
	ret := &internalProtocol.ForceOffline{}
	ret.SessionId = user.SessionId
	ret.UserId = strconv.Itoa(user.Id)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_ForceOffline
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
