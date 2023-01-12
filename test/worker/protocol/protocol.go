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

type Login struct {
	Username string
	Password string
}
