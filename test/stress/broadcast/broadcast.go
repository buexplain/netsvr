// Package broadcast 对网关广播进行压测
package broadcast

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
		ws.OnMessage[protocol.RouterBroadcast] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
	})
}

// Send 广播一条数据
func (r *pool) Send() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterBroadcast, map[string]interface{}{"message": "我是一条按uniqId广播的信息"})
	}
}
