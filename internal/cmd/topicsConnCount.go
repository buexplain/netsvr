package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicsConnCount 获取网关中的某几个主题的连接数
func TopicsConnCount(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.TopicsConnCountReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.TopicsConnCountReq error: %v", err)
		return
	}
	items := map[string]int32{}
	session.Topics.CountConn(payload.Topics, payload.GetAll, items)
	data := &protocol.TopicsConnCountResp{}
	data.ReCtx = payload.ReCtx
	data.Items = items
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_TopicsConnCount
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
