package business

import (
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"netsvr/configs"
	"netsvr/internal/customer/manager"
	"netsvr/pkg/quit"
	"time"
)

type Payload struct {
	SessionId uint32
	Data      []byte
}

func NewPayload(sessionId uint32, data []byte) *Payload {
	return &Payload{SessionId: sessionId, Data: data}
}

type catapult struct {
	ch chan *Payload
}

func (r *catapult) Put(payload *Payload) {
	select {
	case <-quit.Ctx.Done():
		//进程即将停止，不再受理新的数据
		return
	default:
		r.ch <- payload
	}
}

func (r *catapult) write(payload *Payload) {
	conn := manager.Manager.Get(payload.SessionId)
	if conn != nil {
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			logging.Debug("Catapult write message error: %v", err)
		}
	}
}

func (r *catapult) done(number int) {
	//处理通道中的剩余数据
	empty := 0
	for {
		//所有协程遇到多次没拿到数据的情况，视为通道中没有数据了
		if empty > 5 {
			break
		}
		select {
		case v := <-r.ch:
			r.write(v)
		default:
			empty++
		}
	}
	//留下0号协程，进行一个超时等待处理
	if number != 0 {
		return
	}
	//再次处理通道中的剩余数据，直到超时退出
	for {
		select {
		case v := <-r.ch:
			r.write(v)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func (r *catapult) consumer(number int) {
	defer func() {
		quit.Wg.Done()
		if err := recover(); err != nil {
			logging.Error("%v", err)
			time.Sleep(5 * time.Second)
			quit.Wg.Add(1)
			go r.consumer(number)
		}
	}()
	for {
		select {
		case v := <-r.ch:
			r.write(v)
		case <-quit.Ctx.Done():
			//进程即将停止，处理通道中剩余数据
			r.done(number)
			return
		}
	}
}

func (r *catapult) CountWaitSend() int {
	return len(r.ch)
}

// Catapult 把数据发送websocket连接
var Catapult *catapult

func init() {
	Catapult = &catapult{ch: make(chan *Payload, configs.Config.CatapultChanCap)}
	for i := 0; i < configs.Config.CatapultConsumer; i++ {
		quit.Wg.Add(1)
		go Catapult.consumer(i)
	}
}
