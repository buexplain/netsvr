// Package silent 对网关连接负载进行压测
package silent

import (
	"golang.org/x/time/rate"
	"netsvr/pkg/quit"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/wsClient"
	"netsvr/test/stress/internal/wsPool"
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
	r.Pool.AddWebsocket(func(_ *wsClient.Client) {
	})
}

func Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if !configs.Config.Silent.Enable {
		return
	}
	for key, step := range configs.Config.Silent.Step {
		if step.ConnNum <= 0 {
			continue
		}
		l := rate.NewLimiter(rate.Limit(step.ConnectNum), step.ConnectNum)
		for i := 0; i < step.ConnNum; i++ {
			if err := l.Wait(quit.Ctx); err != nil {
				break
			}
			select {
			case <-quit.Ctx.Done():
				break
			default:
				Pool.AddWebsocket()
			}
		}
		log.Logger.Info().Msgf("current silent online %d", Pool.Len())
		if key < len(configs.Config.Silent.Step)-1 && step.Suspend > 0 {
			time.Sleep(time.Duration(step.Suspend) * time.Second)
		}
	}
	go func() {
		<-quit.Ctx.Done()
		Pool.Close()
	}()
}
