package cmd

import (
	"encoding/json"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/log"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/protocol"
	businessUtils "netsvr/test/utils"
)

type forceOfflineGuest struct{}

var ForceOfflineGuest = forceOfflineGuest{}

func (r forceOfflineGuest) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterForceOfflineGuestForUniqId, r.ForUniqId)
}

// ForceOfflineGuestForUniqIdParam 将某个没有session值的连接强制关闭
type ForceOfflineGuestForUniqIdParam struct {
	UniqId string `json:"uniqId"`
	Delay  int32  `json:"delay"`
}

// ForUniqId 将某个没有session值的连接强制关闭
func (forceOfflineGuest) ForUniqId(_ *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ForceOfflineGuestForUniqIdParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse ForceOfflineGuestForUniqIdParam failed")
		return
	}
	ret := &internalProtocol.ForceOfflineGuest{}
	ret.UniqIds = []string{payload.UniqId}
	ret.Delay = payload.Delay
	ret.Data = businessUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "游客您好，您已被迫下线！"})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_ForceOfflineGuest
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
