package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/utils"
)

// CheckOnline 检查某几个连接是否在线
func CheckOnline(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.CheckOnlineResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal connClose.CheckOnlineResp error:%v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if payload.ReCtx.Cmd != int32(protocol.RouterCheckOnlineForUniqId) {
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := utils.BytesToReadOnlyString(payload.ReCtx.Data)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = utils.NewResponse(protocol.RouterNetSvrStatus, map[string]interface{}{"code": 0, "message": "检查某几个连接是否在线成功", "data": payload.UniqIds})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
