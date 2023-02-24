package main

import (
	"github.com/lesismal/nbio/logging"
	"netsvr/pkg/quit"
	"netsvr/test/stress/sign"
	"time"
)

func main() {
	//sign.Pool.In()
	//sign.Pool.Out()
	go func() {
		//return
		//每秒执行一次
		tc := time.NewTicker(time.Second * 1)
		defer tc.Stop()
		for {
			<-tc.C
			logging.Info("当前 %d 个连接", sign.Pool.Len())
			for i := 0; i < 1000; i++ {
				go func() {
					if sign.Pool.Len() < 1000 {
						sign.Pool.In()
					}
					sign.Pool.Out()
				}()
			}
		}
	}()
	select {
	case <-quit.ClosedCh:
		sign.Pool.Close()
		time.Sleep(3 * time.Second)
	}
}
