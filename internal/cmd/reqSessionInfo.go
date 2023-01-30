package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/reqSessionInfo"
	"netsvr/internal/protocol/toWorker/respSessionInfo"
	toWorkerRouter "netsvr/internal/protocol/toWorker/router"
	workerManager "netsvr/internal/worker/manager"
)

// ReqSessionInfo 根据session id获取网关中的用户信息
func ReqSessionInfo(param []byte, processor *workerManager.ConnProcessor) {
	req := reqSessionInfo.ReqSessionInfo{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal reqSessionInfo.ReqSessionInfo error: %v", err)
		return
	}
	data := &respSessionInfo.RespSessionInfo{}
	data.SessionId = req.SessionId
	data.ReCtx = req.ReCtx
	wsConn := customerManager.Manager.Get(req.SessionId)
	if wsConn != nil {
		if info, ok := wsConn.Session().(*session.Info); ok {
			info.GetToRespSessionInfo(data)
		}
	}
	route := &toWorkerRouter.Router{}
	route.Cmd = toWorkerRouter.Cmd_RespSessionInfo
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	processor.Send(b)
}
