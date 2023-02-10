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

// ForceOfflineForUserId 强制关闭某个用户的连接
func ForceOfflineForUserId(_ *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := protocol.ForceOfflineForUserId{}
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse protocol.ForceOfflineForUserId request error: %v", err)
		return
	}
	user := userDb.Collect.GetUserById(payload.UserId)
	if user == nil || user.IsOnline == false {
		return
	}
	ret := &internalProtocol.ForceOffline{}
	ret.UniqId = strconv.Itoa(user.Id)
	ret.PreventConnCloseCmdTransfer = true
	ret.Data = workerUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_ForceOffline
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ForceOfflineForUniqId 强制关闭某个用户的连接
func ForceOfflineForUniqId(_ *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := protocol.ForceOfflineForUniqId{}
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse protocol.ForceOfflineForUniqId request error: %v", err)
		return
	}
	ret := &internalProtocol.ForceOffline{}
	ret.UniqId = payload.UniqId
	ret.PreventConnCloseCmdTransfer = true
	ret.Data = workerUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_ForceOffline
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
