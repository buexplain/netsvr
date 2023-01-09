package business

import (
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/protocol/toServer/singleCast"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"time"
)

type singleCastBusiness struct {
	ch chan *singleCast.SingleCast
}

func (r *singleCastBusiness) execute(singleCast *singleCast.SingleCast) {
	conn := manager.Manager.Get(singleCast.SessionId)
	if conn != nil {
		if err := conn.WriteMessage(websocket.TextMessage, singleCast.Data); err != nil {
			logging.Debug("Error singleCast: %#v", err)
		}
	}
}

func (r *singleCastBusiness) done() {
	//处理通道中的剩余数据
	for {
		for i := len(r.ch); i > 0; i-- {
			v := <-r.ch
			r.execute(v)
		}
		if len(r.ch) == 0 {
			break
		}
	}
	//再次处理通道中的剩余数据，直到超时退出
	for {
		select {
		case v := <-r.ch:
			r.execute(v)
		case <-time.After(3 * time.Second):
			return
		}
	}
}

func (r *singleCastBusiness) consumer(number int) {
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
			r.execute(v)
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

func (r *singleCastBusiness) Send(singleCast *singleCast.SingleCast) {
	select {
	case <-quit.Ctx.Done():
		//进程即将停止，不再受理新的数据
		return
	default:
		r.ch <- singleCast
	}
}

var SingleCast *singleCastBusiness

func init() {
	SingleCast = &singleCastBusiness{ch: make(chan *singleCast.SingleCast, 100)}
	for i := 0; i < 10; i++ {
		quit.Wg.Add(1)
		go SingleCast.consumer(i)
	}
}
