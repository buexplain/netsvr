// Package silent 对网关连接负载进行压测
package silent

import (
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
	r.Pool.AddWebsocket(func(_ *wsClient.Client) {
	})
}
