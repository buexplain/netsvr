package business

import (
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/internal/protocol/toServer/subscribe"
)

// Subscribe 订阅
func Subscribe(subscribe *subscribe.Subscribe) {
	if subscribe.SessionId == 0 || len(subscribe.Topics) == 0 {
		return
	}
	conn := manager.Manager.Get(subscribe.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	//将订阅信息登记到session的info对象里面
	info.Subscribe(subscribe.Topics)
	//主题管理里面也登记上对应的关系
	session.Topics.Set(subscribe.Topics, subscribe.SessionId)
	//订阅成功后，有信息要传递给用户，则转发数据给到用户
	if len(subscribe.Data) > 0 {
		Catapult.Put(NewPayload(subscribe.SessionId, subscribe.Data))
	}
}
