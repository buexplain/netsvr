package cmd

import (
	"encoding/json"
	"google.golang.org/protobuf/proto"
	netsvrProtocol "netsvr/pkg/protocol"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
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
func (forceOfflineGuest) ForUniqId(_ *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ForceOfflineGuestForUniqIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse ForceOfflineGuestForUniqIdParam failed")
		return
	}
	ret := &netsvrProtocol.ForceOfflineGuest{}
	ret.UniqIds = []string{payload.UniqId}
	ret.Delay = payload.Delay
	ret.Data = testUtils.NewResponse(protocol.RouterRespConnClose, map[string]interface{}{"code": 0, "message": "游客您好，您已被迫下线！"})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_ForceOfflineGuest
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
