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

// TopicsUniqIdCount 处理worker发送过来的主题的连接数
func TopicsUniqIdCount(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.TopicsUniqIdCountResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TopicsUniqIdCountResp error: %v", err)
		return
	}
	fmt.Println(payload.String())
	//不是客户端请求网关数据，则忽略
	if payload.ReCtx.Cmd != int32(protocol.RouterTopicsUniqIdCount) {
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
	ret.Data = utils.NewResponse(protocol.RouterTopicsUniqIdCount, map[string]interface{}{"code": 0, "message": "获取网关中主题的连接数成功", "data": payload.Items})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
