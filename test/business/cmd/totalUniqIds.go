package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	businessUtils "netsvr/test/business/utils"
)

type totalUniqIds struct{}

var TotalUniqIds = totalUniqIds{}

func (r totalUniqIds) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterTotalUniqIds, r.Request)
	processor.RegisterWorkerCmd(protocol.RouterTotalUniqIds, r.Response)
}

// Request 获取网关所有的uniqId
func (totalUniqIds) Request(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.TotalUniqIdsReq{}
	req.RouterCmd = int32(protocol.RouterTotalUniqIds)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TotalUniqIds
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// Response 处理worker发送过来的网关所有的uniqId
func (totalUniqIds) Response(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.TotalUniqIdsResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TotalUniqIdsResp error: %v", err)
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
	msg := map[string]interface{}{
		"totalUniqIds": payload.UniqIds,
	}
	ret.Data = businessUtils.NewResponse(protocol.RouterTotalUniqIds, map[string]interface{}{"code": 0, "message": "获取网关所有的uniqId成功", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
