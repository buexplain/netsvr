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

type limit struct{}

var Limit = limit{}

func (r limit) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterLimit, r.Request)
	processor.RegisterWorkerCmd(protocol.RouterLimit, r.Response)
}

// LimitUpdateParam 更新网关中限流配置
type LimitUpdateParam struct {
	WorkerIds []int32 `json:"workerIds"`
	Num       int32   `json:"num"`
}

// Request 更新限流配置、获取网关中的限流配置的真实情况
func (limit) Request(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(LimitUpdateParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse LimitUpdateParam failed")
		return
	}
	req := internalProtocol.LimitReq{}
	req.RouterCmd = int32(protocol.RouterLimit)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Items = []*internalProtocol.LimitUpdateItem{
		{
			Num:       payload.Num,
			WorkerIds: payload.WorkerIds,
		},
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Limit
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// Response 处理worker发送过来的限流配置的真实情况
func (limit) Response(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.LimitResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.LimitResp failed")
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
	ret.Data = businessUtils.NewResponse(protocol.RouterLimit, map[string]interface{}{"code": 0, "message": "获取网关中的限流配置的真实情况成功", "data": payload.Items})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
