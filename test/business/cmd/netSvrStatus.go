package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	businessUtils "netsvr/test/business/utils"
)

type netSvrStatus struct {
}

var NetSvrStatus = netSvrStatus{}

func (r netSvrStatus) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterNetSvrStatus, r.Request)
	processor.RegisterWorkerCmd(protocol.RouterNetSvrStatus, r.Response)
}

// Request 获取网关的信息
func (netSvrStatus) Request(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.NetSvrStatusReq{}
	req.RouterCmd = int32(protocol.RouterNetSvrStatus)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_NetSvrStatus
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// Response 处理worker发送过来的网关状态信息
func (netSvrStatus) Response(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.NetSvrStatusResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.NetSvrStatusResp error: %v", err)
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
	ret.Data = businessUtils.NewResponse(protocol.RouterNetSvrStatus, map[string]interface{}{"code": 0, "message": "获取网关状态信息成功", "data": &payload})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
