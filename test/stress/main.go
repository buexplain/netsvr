package main

import (
	"github.com/lesismal/nbio/logging"
	"netsvr/pkg/quit"
	"netsvr/test/stress/sign"
	"time"
)

func main() {
	logging.SetLevel(logging.LevelInfo)
	//开一批连接上去
	for i := 0; i < 3000; i++ {
		sign.Pool.AddWebsocket()
	}
	go func() {
		//每秒执行一次登录登出操作
		tc := time.NewTicker(time.Second * 1)
		defer tc.Stop()
		for {
			<-tc.C
			logging.Info("当前 %d 个连接", sign.Pool.Len())
			go func() {
				sign.Pool.In()
				sign.Pool.Out()
			}()
		}
	}()
	select {
	case <-quit.ClosedCh:
		sign.Pool.Close()
	}
}
