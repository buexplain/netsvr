package protocol

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
)

// 客户端Cmd路由
const (
	// RouterRespConnOpen 响应连接成功信息
	RouterRespConnOpen = iota + 1
	// RouterLogin 登录
	RouterLogin
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
