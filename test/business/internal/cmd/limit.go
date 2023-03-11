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
func (limit) Request(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(LimitUpdateParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse LimitUpdateParam failed")
		return
	}
	req := netsvrProtocol.LimitReq{}
	req.RouterCmd = int32(protocol.RouterLimit)
	req.CtxData = testUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Items = []*netsvrProtocol.LimitUpdateItem{
		{
			Num:       payload.Num,
			WorkerIds: payload.WorkerIds,
		},
	}
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_Limit
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// Response 处理worker发送过来的限流配置的真实情况
func (limit) Response(param []byte, processor *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.LimitResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.LimitResp failed")
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := testUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = testUtils.NewResponse(protocol.RouterLimit, map[string]interface{}{"code": 0, "message": "获取网关中的限流配置的真实情况成功", "data": payload.Items})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
