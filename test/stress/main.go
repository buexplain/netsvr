package main

import (
	"netsvr/pkg/quit"
	"netsvr/test/stress/multicast"
	"netsvr/test/stress/sign"
	"netsvr/test/stress/singleCast"
	"time"
)

func main() {
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
			go func() {
				sign.Pool.In()
				sign.Pool.Out()
			}()
		}
	}()
	//开一批连接上去
	for i := 0; i < 3000; i++ {
		singleCast.Pool.AddWebsocket()
	}
	go func() {
		//每秒执行一次单播操作
		tc := time.NewTicker(time.Second * 1)
		defer tc.Stop()
		for {
			<-tc.C
			go func() {
				singleCast.Pool.Send()
			}()
		}
	}()
	//开一批连接上去
	for i := 0; i < 3000; i++ {
		multicast.Pool.AddWebsocket()
	}
	go func() {
		//每秒执行一次组播操作
		tc := time.NewTicker(time.Second * 1)
		defer tc.Stop()
		for {
			<-tc.C
			go func() {
				multicast.Pool.Send()
			}()
		}
	}()
	select {
	case <-quit.ClosedCh:
		sign.Pool.Close()
		singleCast.Pool.Close()
	}
}
