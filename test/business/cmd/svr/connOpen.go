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
	//构造单播数据
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = payload.UniqId
	ret.Data = utils.NewResponse(protocol.RouterRespConnOpen, map[string]interface{}{"code": 0, "message": fmt.Sprintf("连接网关成功，uniqId: %s", payload.UniqId)})
	//发送到网关
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
