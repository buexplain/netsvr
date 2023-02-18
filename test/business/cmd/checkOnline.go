package cmd

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	businessUtils "netsvr/test/business/utils"
)

type checkOnline struct{}

var CheckOnline = checkOnline{}

func (r checkOnline) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterCheckOnlineForUniqId, r.RequestForUniqId)
	processor.RegisterWorkerCmd(protocol.RouterCheckOnlineForUniqId, r.ResponseForUniqId)
}

// CheckOnlineForUniqIdParam 检查某几个连接是否在线
type CheckOnlineForUniqIdParam struct {
	UniqIds []string `json:"uniqIds"`
}

// RequestForUniqId 向worker发起请求，检查某几个连接是否在线
func (checkOnline) RequestForUniqId(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := CheckOnlineForUniqIdParam{}
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse CheckOnlineForUniqIdParam error: %v", err)
		return
	}
	req := &internalProtocol.CheckOnlineReq{}
	req.RouterCmd = int32(protocol.RouterCheckOnlineForUniqId)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	req.UniqIds = payload.UniqIds
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_CheckOnline
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseForUniqId 处理worker的响应
func (checkOnline) ResponseForUniqId(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.CheckOnlineResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal connClose.CheckOnlineResp error:%v", err)
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := businessUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = businessUtils.NewResponse(protocol.RouterNetSvrStatus, map[string]interface{}{"code": 0, "message": "检查某几个连接是否在线成功", "data": payload.UniqIds})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
