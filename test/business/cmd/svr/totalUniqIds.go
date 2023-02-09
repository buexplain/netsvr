package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/utils"
)

// TotalUniqIds 处理worker发送过来的网关所有的uniqId
func TotalUniqIds(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.TotalUniqIdsResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TotalUniqIdsResp error: %v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if payload.ReCtx.Cmd != int32(protocol.RouterTotalUniqIds) {
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := utils.BytesToReadOnlyString(payload.ReCtx.Data)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = uniqId
	msg := map[string]interface{}{
		"totalUniqIds": payload.UniqIds,
	}
	ret.Data = utils.NewResponse(protocol.RouterTotalUniqIds, map[string]interface{}{"code": 0, "message": "获取网关所有的uniqId成功", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
