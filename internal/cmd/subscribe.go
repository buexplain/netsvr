package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/subscribe"
	workerManager "netsvr/internal/worker/manager"
)

// Subscribe 订阅
func Subscribe(param []byte, _ *workerManager.ConnProcessor) {
	req := subscribe.Subscribe{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal subscribe.Subscribe error: %v", err)
		return
	}
	if req.SessionId == 0 || len(req.Topics) == 0 {
		return
	}
	conn := manager.Manager.Get(req.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	//将订阅信息登记到session的info对象里面
	info.Subscribe(req.Topics)
	//主题管理里面也登记上对应的关系
	session.Topics.Set(req.Topics, req.SessionId)
	//订阅成功后，有信息要传递给用户，则转发数据给到用户
	if len(req.Data) > 0 {
		Catapult.Put(NewPayload(req.SessionId, req.Data))
	}
}
