package protocol

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
)

// 客户端Cmd路由
const (
	// RouterRespConnOpen 响应连接成功信息
	RouterRespConnOpen = iota + 1
	// RouterNetSvrStatus 获取网关的信息
	RouterNetSvrStatus
	// RouterTotalSessionId 获取网关所有在线的session id
	RouterTotalSessionId
	// RouterLogin 登录
	RouterLogin
	//RouterLogout 退出登录
	RouterLogout
	// RouterUpdateSessionUserInfo 更新网关中的用户信息
	RouterUpdateSessionUserInfo
	// RouterSingleCast 单播给某个用户
	RouterSingleCast
	// RouterMulticast 组播给多个用户
	RouterMulticast
	// RouterBroadcast 广播给所有用户
	RouterBroadcast
	// RouterSubscribe 订阅
	RouterSubscribe
	// RouterUnsubscribe 取消订阅
	RouterUnsubscribe
	// RouterTopicsConnCount 获取网关中的某几个主题的连接数
	RouterTopicsConnCount
	// RouterTopicsSessionId 获取网关中的某几个主题的连接session id
	RouterTopicsSessionId
	// RouterPublish 发布信息
	RouterPublish
)

type ClientRouter struct {
	Cmd  int
	Data string
}

func ParseClientRouter(data []byte) *ClientRouter {
	ret := new(ClientRouter)
	if err := json.Unmarshal(data, ret); err != nil {
		logging.Debug("Parse client router error: %v", err)
		return nil
	}
	return ret
}

// Login 客户端发送的登录信息
type Login struct {
	Username string
	Password string
}

// SingleCast 客户端发送的单播信息
type SingleCast struct {
	Message string
	UserId  int `json:"userId"`
}

// Multicast 客户端发送的组播信息
type Multicast struct {
	Message string
	UserIds []int `json:"userIds"`
}

// Broadcast 客户端发送的广播信息
type Broadcast struct {
	Message string
}

// Subscribe 客户端发送的订阅信息
type Subscribe struct {
	Topics []string
}

// Unsubscribe 客户端发送的取消订阅信息
type Unsubscribe struct {
	Topics []string
}

// TopicsSessionId 获取网关中的某几个主题的连接session id
type TopicsSessionId struct {
	Topics []string
}

// TopicsConnCount 获取网关中的某几个主题的连接数
type TopicsConnCount struct {
	GetAll bool
	Topics []string
}

// Publish 客户端发送的发布信息
type Publish struct {
	Message string
	Topic   string
}
