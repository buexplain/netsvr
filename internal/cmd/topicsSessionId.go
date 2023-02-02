package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// TopicsSessionId 获取网关中的某几个主题的session id
func TopicsSessionId(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.TopicsSessionIdReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.TopicsSessionId error: %v", err)
		return
	}
	items := map[string]string{}
	session.Topics.Gets(payload.Topics, items)
	data := &protocol.TopicsSessionIdResp{}
	data.ReCtx = payload.ReCtx
	data.Items = items
	route := &protocol.Router{}
	route.Cmd = protocol.Cmd_TopicsSessionId
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
