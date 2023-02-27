package multicast

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
	ws.OnMessage[protocol.RouterMulticastForUniqId] = func(payload gjson.Result) {
		logging.Debug(payload.Raw)
	}
	ws.OnClose = func() {
	}
	go ws.LoopSend()
	go ws.LoopRead()
	r.m.Lock()
	defer r.m.Unlock()
	r.p["sign"+utils.UniqId()] = ws
}

// Send 组播一条数据
func (r *pool) Send() {
	r.m.RLock()
	defer r.m.RUnlock()
	var ws *wsClient.Client
	uniqIds := make([]string, 0, 100)
	replaceIndex := 0
	for _, ws = range r.p {
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

//multicast
//001{"cmd":19,"data":"{\"message\":\"我是一条按uniqId组播的信息\",\"uniqIds\":[\"0063FC645739C4E6B1iy\",\"222\"]}"}
