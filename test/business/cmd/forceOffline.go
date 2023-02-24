package cmd

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/userDb"
	"netsvr/test/protocol"
	businessUtils "netsvr/test/utils"
	"strconv"
)

type forceOffline struct{}

var ForceOffline = forceOffline{}

func (r forceOffline) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterForceOfflineForUserId, r.ForUserId)
	processor.RegisterBusinessCmd(protocol.RouterForceOfflineForUniqId, r.ForUniqId)
}

// ForceOfflineForUserIdParam 强制踢下线某个用户
type ForceOfflineForUserIdParam struct {
	UserId int `json:"userId"`
}

// ForUserId 强制关闭某个用户的连接
func (forceOffline) ForUserId(_ *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ForceOfflineForUserIdParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse ForceOfflineForUserIdParam error: %v", err)
		return
	}
	user := userDb.Collect.GetUserById(payload.UserId)
	if user == nil || user.IsOnline == false {
		return
	}
	ret := &internalProtocol.ForceOffline{}
	ret.UniqId = strconv.Itoa(user.Id)
	ret.PreventConnCloseCmdTransfer = true
	ret.Data = businessUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_ForceOffline
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ForceOfflineForUniqIdParam 强制踢下线某个用户
type ForceOfflineForUniqIdParam struct {
	UniqId string `json:"uniqId"`
}

// ForUniqId 强制关闭某个用户的连接
func (forceOffline) ForUniqId(_ *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ForceOfflineForUniqIdParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse ForceOfflineForUniqIdParam error: %v", err)
		return
	}
	ret := &internalProtocol.ForceOffline{}
	ret.UniqId = payload.UniqId
	ret.PreventConnCloseCmdTransfer = true
	ret.Data = businessUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "您已被迫下线！"})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_ForceOffline
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
