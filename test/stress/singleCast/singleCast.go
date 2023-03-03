package singleCast

import (
	"fmt"
	"github.com/tidwall/gjson"
	"netsvr/configs"
	"netsvr/internal/log"
	"netsvr/test/protocol"
	"netsvr/test/utils/wsClient"
	"sync"
	"time"
)

var Pool *pool

type pool struct {
	p map[string]*wsClient.Client
	m sync.RWMutex
}

func init() {
	Pool = &pool{p: map[string]*wsClient.Client{}, m: sync.RWMutex{}}
	go func() {
		tc := time.NewTicker(time.Second * 30)
		defer tc.Stop()
		for {
			<-tc.C
			Pool.m.RLock()
			for _, ws := range Pool.p {
				ws.Heartbeat()
			}
			Pool.m.RUnlock()
		}
	}()
}

func (r *pool) Len() int {
	r.m.RLock()
	defer r.m.RUnlock()
	return len(r.p)
}

func (r *pool) Close() {
	var ws *wsClient.Client
	for {
		r.m.RLock()
		for _, ws = range r.p {
			break
		}
		r.m.RUnlock()
		if ws == nil {
			break
		}
		ws.Close()
		ws = nil
	}
}

func (r *pool) AddWebsocket() {
	ws := wsClient.New(fmt.Sprintf("ws://%s%s", configs.Config.CustomerListenAddress, configs.Config.CustomerHandlePattern))
	if ws == nil {
		return
	}
	ws.OnMessage[protocol.RouterSingleCastForUniqId] = func(payload gjson.Result) {
		log.Logger.Debug().Msg(payload.Raw)
	}
	ws.OnClose = func() {
		r.m.Lock()
		defer r.m.Unlock()
		delete(r.p, ws.LocalUniqId)
	}
	go ws.LoopSend()
	go ws.LoopRead()
	r.m.Lock()
	defer r.m.Unlock()
	r.p[ws.LocalUniqId] = ws
}

// Send 单播一条数据
func (r *pool) Send() {
	r.m.RLock()
	defer r.m.RUnlock()
	var ws *wsClient.Client
	var preWs *wsClient.Client
	for _, ws = range r.p {
		if preWs == nil {
			preWs = ws
		}
		ws.Send(protocol.RouterSingleCastForUniqId, map[string]string{"message": "我是一条按uniqId单播的信息", "uniqId": preWs.UniqId})
		preWs = ws
	}
}
