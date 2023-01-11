package business

import (
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
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
			logging.Debug("Error singleCast: %#v", err)
		}
	}
}

func (r *catapult) done() {
	//处理通道中的剩余数据
	for {
		for i := len(r.ch); i > 0; i-- {
			v := <-r.ch
			r.write(v)
		}
		if len(r.ch) == 0 {
			break
		}
	}
	//再次处理通道中的剩余数据，直到超时退出
	for {
		select {
		case v := <-r.ch:
			r.write(v)
		case <-time.After(3 * time.Second):
			return
		}
	}
}

func (r *catapult) consumer(number int) {
	defer func() {
		quit.Wg.Done()
		if err := recover(); err != nil {
			logging.Error("%#v", err)
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
			if number == 0 {
				//紧保留一个协程处理通道中剩余的数据，尽量保证工作进程的数据转发到用户
				r.done()
			}
			return
		}
	}
}

// Catapult 把数据发送websocket连接
var Catapult *catapult

func init() {
	Catapult = &catapult{ch: make(chan *Payload, 100)}
	for i := 0; i < 10; i++ {
		quit.Wg.Add(1)
		go Catapult.consumer(i)
	}
}
