// Package singleCast 对网关单播进行压测
package singleCast

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
		ws.OnMessage[protocol.RouterSingleCastForUniqId] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
	})
}

// Send 单播一条数据
func (r *pool) Send() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	var preWs *wsClient.Client
	for _, ws = range r.P {
		if preWs == nil {
			preWs = ws
		}
		ws.Send(protocol.RouterSingleCastForUniqId, map[string]string{"message": "我是一条按uniqId单播的信息", "uniqId": preWs.UniqId})
		preWs = ws
	}
}
