package sign

import (
	"fmt"
	"github.com/lesismal/nbio/logging"
	"github.com/tidwall/gjson"
	"netsvr/configs"
	"netsvr/pkg/utils"
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
	r.m.Lock()
	defer r.m.Unlock()
	for _, ws := range r.p {
		ws.Close()
	}
}

func (r *pool) AddWebsocket() {
	ws := wsClient.New(fmt.Sprintf("ws://%s%s", configs.Config.CustomerListenAddress, configs.Config.CustomerHandlePattern))
	if ws == nil {
		return
	}
	ws.OnMessage[protocol.RouterSignInForForge] = func(payload gjson.Result) {
		logging.Debug(payload.Raw)
	}
	ws.OnMessage[protocol.RouterSignOutForForge] = func(payload gjson.Result) {
		logging.Debug(payload.Raw)
	}
	go ws.LoopSend()
	go ws.LoopRead()
	r.m.Lock()
	defer r.m.Unlock()
	r.p["sign"+utils.UniqId()] = ws
}

// In 登录操作
func (r *pool) In() {
	r.m.RLock()
	defer r.m.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.p {
		ws.Send(protocol.RouterSignInForForge, nil)
	}
}

// Out 退出登录操作
func (r *pool) Out() {
	r.m.RLock()
	defer r.m.RUnlock()
	var ws *wsClient.Client
	for _, ws = range r.p {
		ws.Send(protocol.RouterSignOutForForge, nil)
	}
}
