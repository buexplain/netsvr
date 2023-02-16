package catapult

import (
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"netsvr/configs"
	"netsvr/internal/customer/manager"
	"netsvr/pkg/quit"
	"runtime/debug"
	"time"
)

type Payload struct {
	uniqId string
	Data   []byte
}

func (r Payload) GetUniqId() string {
	return r.uniqId
}

func (r Payload) GetData() []byte {
	return r.Data
}

func NewPayload(uniqId string, data []byte) *Payload {
	return &Payload{uniqId: uniqId, Data: data}
}

type PayloadInterface interface {
	GetUniqId() string
	GetData() []byte
}

type catapult struct {
	ch chan PayloadInterface
}

func (r *catapult) Put(payload PayloadInterface) {
	select {
	case <-quit.Ctx.Done():
		//网关进程即将停止，不再受理新的数据
		return
	default:
		r.ch <- payload
	}
}

func (r *catapult) write(payload PayloadInterface) {
	conn := manager.Manager.Get(payload.GetUniqId())
	if conn != nil {
		if err := conn.WriteMessage(websocket.TextMessage, payload.GetData()); err != nil {
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
			logging.Error("Customer catapult consumer coroutine is closed, error: %v\n%s", err, debug.Stack())
			time.Sleep(5 * time.Second)
			quit.Wg.Add(1)
			go r.consumer(number)
		} else {
			logging.Debug("Customer catapult consumer coroutine is closed")
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

func (r *catapult) Len() int {
	return len(r.ch)
}

// Catapult 把数据发送websocket连接
var Catapult *catapult

func init() {
	Catapult = &catapult{ch: make(chan PayloadInterface, configs.Config.CatapultChanCap)}
	for i := 0; i < configs.Config.CatapultConsumer; i++ {
		quit.Wg.Add(1)
		go Catapult.consumer(i)
	}
}
