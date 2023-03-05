// 这个是压测模块
package main

import (
	"fmt"
	"math/rand"
	"netsvr/internal/log"
	"netsvr/pkg/quit"
	"netsvr/test/stress/broadcast"
	"netsvr/test/stress/multicast"
	"netsvr/test/stress/sign"
	"netsvr/test/stress/singleCast"
	"time"
)

const signNum = 100
const singleCastNum = 100
const multicastNum = 20
const broadcastNum = 2

func main() {
	//开一批连接上去
	for i := 0; i < signNum; i++ {
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
				if rand.Intn(10) > 5 {
					time.Sleep(time.Millisecond * 100)
				}
				sign.Pool.Out()
			}()
		}
	}()
	//开一批连接上去
	for i := 0; i < singleCastNum; i++ {
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
	for i := 0; i < multicastNum; i++ {
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
	//开一批连接上去
	for i := 0; i < broadcastNum; i++ {
		broadcast.Pool.AddWebsocket()
	}
	go func() {
		//每秒执行一次组播操作
		tc := time.NewTicker(time.Second * 1)
		defer tc.Stop()
		for {
			<-tc.C
			go func() {
				broadcast.Pool.Send()
			}()
		}
	}()
	go func() {
		tc := time.NewTicker(time.Second * 20)
		defer tc.Stop()
		for {
			<-tc.C
			online := sign.Pool.Len() + singleCast.Pool.Len() + multicast.Pool.Len() + broadcast.Pool.Len()
			fmt.Printf("current online %d\n", online)
			log.Logger.Info().Msgf("current online %d", online)
		}
	}()
	select {
	case <-quit.ClosedCh:
		sign.Pool.Close()
		singleCast.Pool.Close()
		multicast.Pool.Close()
	}
}
