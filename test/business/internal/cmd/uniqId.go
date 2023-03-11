package cmd

import (
	"google.golang.org/protobuf/proto"
	netsvrProtocol "netsvr/pkg/protocol"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
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
func (uniqId) RequestList(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &netsvrProtocol.UniqIdListReq{}
	req.RouterCmd = int32(protocol.RouterUniqIdList)
	req.CtxData = testUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_UniqIdList
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseList 处理worker发送过来的网关所有的uniqId
func (uniqId) ResponseList(param []byte, processor *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.UniqIdListResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.UniqIdListResp failed")
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := testUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = uniqId
	msg := map[string]interface{}{
		"uniqIds": payload.UniqIds,
	}
	ret.Data = testUtils.NewResponse(protocol.RouterUniqIdList, map[string]interface{}{"code": 0, "message": "获取网关所有的uniqId成功", "data": msg})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// RequestCount 获取网关中uniqId的数量
func (uniqId) RequestCount(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &netsvrProtocol.UniqIdCountReq{}
	req.RouterCmd = int32(protocol.RouterUniqIdCount)
	req.CtxData = testUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_UniqIdCount
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseCount 处理worker发送过来的网关中uniqId的数量
func (uniqId) ResponseCount(param []byte, processor *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.UniqIdCountResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.UniqIdCountResp failed")
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := testUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = uniqId
	msg := map[string]interface{}{
		"count": payload.Count,
	}
	ret.Data = testUtils.NewResponse(protocol.RouterUniqIdCount, map[string]interface{}{"code": 0, "message": "获取网关中uniqId的数量成功", "data": msg})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
