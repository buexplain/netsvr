// Package sign 对网关登录登出进行压测
package sign

import (
	"github.com/tidwall/gjson"
	"netsvr/internal/log"
	"netsvr/test/protocol"
	"netsvr/test/utils/wsClient"
	"netsvr/test/utils/wsPool"
	"sync"
)

var Pool *pool

type pool struct {
	wsPool.Pool
}

func init() {
	Pool = &pool{}
	Pool.P = map[string]*wsClient.Client{}
	Pool.Mux = &sync.RWMutex{}
	go Pool.Heartbeat()
}

func (r *pool) AddWebsocket() {
	r.Pool.AddWebsocket(func(ws *wsClient.Client) {
		ws.OnMessage[protocol.RouterSignInForForge] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
		ws.OnMessage[protocol.RouterSignOutForForge] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
	})
}

// In 登录操作
func (r *pool) In() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterSignInForForge, nil)
	}
}

// Out 退出登录操作
func (r *pool) Out() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterSignOutForForge, nil)
	}
}
