package wsPool

import (
	"fmt"
	"netsvr/configs"
	"netsvr/test/utils/wsClient"
	"sync"
	"time"
)

type Pool struct {
	P   map[string]*wsClient.Client
	Mux *sync.RWMutex
}

func (r *Pool) Heartbeat() {
	tc := time.NewTicker(time.Second * 30)
	defer tc.Stop()
	for {
		<-tc.C
		r.Mux.RLock()
		for _, ws := range r.P {
			ws.Heartbeat()
		}
		r.Mux.RUnlock()
	}
}

func (r *Pool) Len() int {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	return len(r.P)
}

func (r *Pool) Close() {
	var ws *wsClient.Client
	for {
		r.Mux.RLock()
		for _, ws = range r.P {
			break
		}
		r.Mux.RUnlock()
		if ws == nil {
			break
		}
		ws.Close()
		ws = nil
	}
}

func (r *Pool) AddWebsocket(option func(ws *wsClient.Client)) {
	ws := wsClient.New(fmt.Sprintf("ws://%s%s", configs.Config.Customer.ListenAddress, configs.Config.Customer.HandlePattern))
	if ws == nil {
		return
	}
	option(ws)
	ws.OnClose = func() {
		r.Mux.Lock()
		defer r.Mux.Unlock()
		delete(r.P, ws.LocalUniqId)
	}
	go ws.LoopSend()
	go ws.LoopRead()
	r.Mux.Lock()
	defer r.Mux.Unlock()
	r.P[ws.LocalUniqId] = ws
}
