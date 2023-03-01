package cmd

import (
	"google.golang.org/protobuf/proto"
	"netsvr/internal/log"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/protocol"
	businessUtils "netsvr/test/utils"
)

type metrics struct{}

var Metrics = metrics{}

func (r metrics) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterMetrics, r.Request)
	processor.RegisterWorkerCmd(protocol.RouterMetrics, r.Response)
}

// Request 获取网关统计的服务状态
func (metrics) Request(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.MetricsReq{}
	req.RouterCmd = int32(protocol.RouterMetrics)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Metrics
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// Response 处理worker发送过来的网关统计的服务状态
func (metrics) Response(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.MetricsResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.MetricsResp failed")
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := businessUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = businessUtils.NewResponse(protocol.RouterMetrics, map[string]interface{}{"code": 0, "message": "获取网关统计的服务状态成功", "data": payload.Items})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
