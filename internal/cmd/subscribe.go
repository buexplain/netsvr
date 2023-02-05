package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// Subscribe 订阅
func Subscribe(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.Subscribe{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.Subscribe error: %v", err)
		return
	}
	if payload.SessionId == 0 || len(payload.Topics) == 0 {
		return
	}
	conn := customerManager.Manager.Get(payload.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	//连接id对应的客户已经被顶了，则不做修改
	if payload.UserId != "" {
		userId := info.GetUserId()
		if userId != "" && userId != payload.UserId {
			return
		}
	}
	//将订阅信息登记到session的info对象里面
	info.Subscribe(payload.Topics)
	//主题管理里面也登记上对应的关系
	session.Topics.Set(payload.Topics, payload.SessionId)
	//订阅成功后，有信息要传递给用户，则转发数据给到用户
	if len(payload.Data) > 0 {
		Catapult.Put(payload)
	}
}
