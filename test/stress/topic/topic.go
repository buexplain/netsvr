// Package topic 对网关主题操作进行压测
package topic

import (
	"github.com/tidwall/gjson"
	"netsvr/internal/log"
	"netsvr/test/protocol"
	businessUtils "netsvr/test/utils"
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
		ws.OnMessage[protocol.RouterTopicSubscribe] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
		ws.OnMessage[protocol.RouterTopicUnsubscribe] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
		ws.InitTopic()
	})
}

// Subscribe 订阅主题
func (r *pool) Subscribe() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		//伪造主题
		topics := make([]string, 0, 20)
		for i := 0; i < 10; i++ {
			topics = append(topics, businessUtils.GetRandStr(2))
		}
		ws.Send(protocol.RouterTopicSubscribe, map[string][]string{"topics": ws.GetSubscribeTopic()})
	}
}

// Unsubscribe 取消订阅主题
func (r *pool) Unsubscribe() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterTopicUnsubscribe, map[string][]string{"topics": ws.GetUnsubscribeTopic()})
	}
}
func (r *pool) Publish() {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.P {
		ws.Send(protocol.RouterTopicPublish, map[string]string{"topic": ws.GetTopic(), "message": "我是一条发布信息"})
	}
}
