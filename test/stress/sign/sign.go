package sign

import (
	"encoding/json"
	"fmt"
	"netsvr/configs"
	"netsvr/test/protocol"
	"netsvr/test/utils/wsClient"
	"strconv"
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
			Pool.m.RUnlock()
			for _, ws := range Pool.p {
				ws.Heartbeat()
			}
		}
	}()
}

type InCmd struct {
	Cmd  int32 `json:"cmd"`
	Data struct {
		Code int32 `json:"code"`
		Data struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			UniqID string `json:"uniqId"`
		} `json:"data"`
		Message string `json:"message"`
	} `json:"data"`
}

func (r *pool) Len() int {
	r.m.RLock()
	defer r.m.RUnlock()
	return len(r.p)
}

func (r *pool) Close() {
	for {
		r.m.RLock()
		var ws *wsClient.Client
		for _, ws = range r.p {
			break
		}
		r.m.RUnlock()
		if ws == nil {
			return
		}
		ws.Close()
	}
}

// In 构造一个登录的ws
func (r *pool) In() {
	ws := wsClient.New(fmt.Sprintf("ws://%s%s", configs.Config.CustomerListenAddress, configs.Config.CustomerHandlePattern))
	if ws == nil {
		return
	}
	ws.OnClose = func() {
		r.m.Lock()
		defer r.m.Unlock()
		delete(r.p, ws.UniqId)
	}
	ch := make(chan struct{})
	ws.OnMessage = func(p []byte) {
		var err error
		ret := InCmd{}
		err = json.Unmarshal(p, &ret)
		if err != nil {
			ws.Close()
			return
		}
		if ret.Cmd != int32(protocol.RouterSignInForForge) || ret.Data.Code != 0 {
			ws.Close()
			return
		}
		ws.UniqId = ret.Data.Data.UniqID
		//登录成功，加入到池子里面
		r.m.Lock()
		defer r.m.Unlock()
		r.p[ws.UniqId] = ws
		close(ch)
	}
	go ws.LoopSend()
	go ws.LoopRead()
	//发送登录指令
	ws.Send([]byte(`001{"cmd":` + strconv.Itoa(int(protocol.RouterSignInForForge)) + `,"data":"{\"password\":\"123456\",\"username\":\"请选择账号\"}"}`))
	<-ch
}

// Out 随机拿出一个ws进行退出登录操作
func (r *pool) Out() {
	r.m.RLock()
	var ws *wsClient.Client
	for _, ws = range r.p {
		break
	}
	r.m.RUnlock()
	if ws == nil {
		return
	}
	ws.Send([]byte(`001{"cmd":` + strconv.Itoa(int(protocol.RouterSignOutForForge)) + `,"data":"{}"}`))
}
