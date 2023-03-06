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
	"netsvr/test/stress/silent"
	"netsvr/test/stress/singleCast"
	"netsvr/test/stress/topic"
	"time"
)

const silentNum = 500
const signNum = 10
const singleCastNum = 50
const multicastNum = 10
const broadcastNum = 1
const topicNum = 10

func main() {
	//开一批连接上去
	x := 0
	for i := 0; i < silentNum; i++ {
		x++
		if x%200 == 0 {
			time.Sleep(time.Second * 1)
		}
		silent.Pool.AddWebsocket()
	}
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
		//每秒执行一次广播操作
		tc := time.NewTicker(time.Second * 1)
		defer tc.Stop()
		for {
			<-tc.C
			go func() {
				broadcast.Pool.Send()
			}()
		}
	}()
	//开一批连接上去
	for i := 0; i < topicNum; i++ {
		topic.Pool.AddWebsocket()
	}
	go func() {
		//每秒执行一次主题订阅、取消订阅、发布操作
		tc := time.NewTicker(time.Second * 1)
		defer tc.Stop()
		for {
			<-tc.C
			go func() {
				topic.Pool.Subscribe()
				if rand.Intn(10) > 5 {
					time.Sleep(time.Millisecond * 100)
				}
				topic.Pool.Unsubscribe()
				topic.Pool.Publish()
			}()
		}
	}()
	go func() {
		tc := time.NewTicker(time.Second * 30)
		defer tc.Stop()
		for {
			<-tc.C
			online := sign.Pool.Len() + singleCast.Pool.Len() + multicast.Pool.Len() + broadcast.Pool.Len() + silent.Pool.Len() + topic.Pool.Len()
			fmt.Printf("%s current online %d\n", time.Now().String(), online)
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
