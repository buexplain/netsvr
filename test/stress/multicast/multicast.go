// Package multicast 对网关组播进行压测
package multicast

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
		ws.OnMessage[protocol.RouterMulticastForUniqId] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
	})
}

// Send 组播一条数据
func (r *pool) Send() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	uniqIds := make([]string, 0, 100)
	replaceIndex := 0
	for _, ws = range r.P {
		if len(uniqIds) < 100 {
			uniqIds = append(uniqIds, ws.UniqId)
		} else {
			uniqIds[replaceIndex] = ws.UniqId
			replaceIndex++
			if replaceIndex == 100 {
				replaceIndex = 0
			}
		}
		ws.Send(protocol.RouterMulticastForUniqId, map[string]interface{}{"message": "我是一条按uniqId组播的信息", "uniqIds": uniqIds})
	}
}
