package svr

import (
	"fmt"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/utils"
)

// ConnOpen 客户端打开连接
func ConnOpen(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.ConnOpen{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal connOpen.ConnOpen error:%v", err)
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	//构造单播数据
	ret := &internalProtocol.SingleCast{}
	ret.SessionId = payload.SessionId
	ret.Data = utils.NewResponse(protocol.RouterRespConnOpen, map[string]interface{}{"code": 0, "message": fmt.Sprintf("连接网关成功，sessionId: %d", payload.SessionId)})
	//将业务数据放到路由上
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
