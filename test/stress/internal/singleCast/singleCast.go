// Package singleCast 对网关单播进行压测
package singleCast

import (
	"github.com/tidwall/gjson"
	"golang.org/x/time/rate"
	"netsvr/pkg/quit"
	"netsvr/test/pkg/protocol"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsPool"
	"strings"
	"sync"
	"time"
)

var Pool *pool

type pool struct {
	wsPool.Pool
}

func init() {
	Pool = &pool{}
	Pool.P = map[string]*wsClient.Client{}
	Pool.Mux = &sync.RWMutex{}
	if configs.Config.Heartbeat > 0 {
		go Pool.Heartbeat()
	}
}

func (r *pool) AddWebsocket() {
	r.Pool.AddWebsocket(func(ws *wsClient.Client) {
		ws.OnMessage[protocol.RouterSingleCastForUniqId] = func(payload gjson.Result) {
			log.Logger.Debug().Msg(payload.Raw)
		}
	})
}

// Send 单播一条数据
func (r *pool) Send(message string) {
	r.Mux.RLock()
	defer r.Mux.RUnlock()
	var ws *wsClient.Client
	var preWs *wsClient.Client
	for _, ws = range r.P {
		if preWs == nil {
			preWs = ws
		}
		ws.Send(protocol.RouterSingleCastForUniqId, map[string]string{"message": message, "uniqId": preWs.UniqId})
		preWs = ws
	}
}

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.SingleCast.Enable {
		return
	}
	if configs.Config.SingleCast.MessageInterval > 0 {
		go func() {
			tc := time.NewTicker(time.Second * time.Duration(configs.Config.SingleCast.MessageInterval))
			defer tc.Stop()
			message := "我是一条按uniqId单播的信息"
			if configs.Config.SingleCast.MessageLen > 0 {
				message = strings.Repeat("a", configs.Config.SingleCast.MessageLen)
			}
			for {
				select {
				case <-tc.C:
					Pool.Send(message)
				case <-quit.Ctx.Done():
					return
				}
			}
		}()
	}
	for key, step := range configs.Config.SingleCast.Step {
		if step.ConnNum <= 0 {
			continue
		}
		l := rate.NewLimiter(rate.Limit(step.ConnectNum), step.ConnectNum)
		for i := 0; i < step.ConnNum; i++ {
			if err := l.Wait(quit.Ctx); err != nil {
				continue
			}
			Pool.AddWebsocket()
		}
		log.Logger.Info().Msgf("current singleCast online %d", Pool.Len())
		if key < len(configs.Config.SingleCast.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	go func() {
		<-quit.Ctx.Done()
		Pool.Close()
	}()
}
