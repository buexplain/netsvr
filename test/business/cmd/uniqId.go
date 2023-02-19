package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	businessUtils "netsvr/test/business/utils"
)

type uniqId struct{}

var UniqId = uniqId{}

func (r uniqId) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterUniqIdList, r.RequestList)
	processor.RegisterWorkerCmd(protocol.RouterUniqIdList, r.ResponseList)

	processor.RegisterBusinessCmd(protocol.RouterUniqIdCount, r.RequestCount)
	processor.RegisterWorkerCmd(protocol.RouterUniqIdCount, r.ResponseCount)
}

// RequestList 获取网关所有的uniqId
func (uniqId) RequestList(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.UniqIdListReq{}
	req.RouterCmd = int32(protocol.RouterUniqIdList)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_UniqIdList
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseList 处理worker发送过来的网关所有的uniqId
func (uniqId) ResponseList(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.UniqIdListResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.UniqIdListResp error: %v", err)
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
		"uniqIds": payload.UniqIds,
	}
	ret.Data = businessUtils.NewResponse(protocol.RouterUniqIdList, map[string]interface{}{"code": 0, "message": "获取网关所有的uniqId成功", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// RequestCount 获取网关中uniqId的数量
func (uniqId) RequestCount(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.UniqIdCountReq{}
	req.RouterCmd = int32(protocol.RouterUniqIdCount)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_UniqIdCount
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseCount 处理worker发送过来的网关中uniqId的数量
func (uniqId) ResponseCount(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.UniqIdCountResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.UniqIdCountResp error: %v", err)
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
		"count": payload.Count,
	}
	ret.Data = businessUtils.NewResponse(protocol.RouterUniqIdCount, map[string]interface{}{"code": 0, "message": "获取网关中uniqId的数量成功", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
