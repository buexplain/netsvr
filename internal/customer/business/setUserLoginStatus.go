package business

import (
	"github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/internal/protocol/toServer/setUserLoginStatus"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"time"
)

type setUserLoginStatusBusiness struct {
	ch chan *setUserLoginStatus.SetUserLoginStatus
}

func (r *setUserLoginStatusBusiness) execute(userLoginStatus *setUserLoginStatus.SetUserLoginStatus) {
	conn := manager.Manager.Get(userLoginStatus.SessionId)
	if conn == nil {
		return
	}
	info, ok := conn.Session().(*session.Info)
	if !ok {
		return
	}
	//设置登录状态
	if userLoginStatus.LoginStatus {
		//登录成功，设置用户的信息
		info.SetLoginStatusOk(userLoginStatus.UserInfo)
	} else {
		//登录失败，重置登录状态为等待登录中，用户可以发起二次登录
		info.SetLoginStatus(session.LoginStatusWait)
	}
	//如果有消息要转告给用户，则转发消息给用户
	if len(userLoginStatus.Data) > 0 {
		if err := conn.WriteMessage(websocket.TextMessage, userLoginStatus.Data); err != nil {
			logging.Debug("Error userLoginStatus: %#v", err)
		}
	}
}

func (r *setUserLoginStatusBusiness) done() {
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

func (r *setUserLoginStatusBusiness) consumer(number int) {
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
			//收到全局的关闭信号
			if number == 0 {
				//紧保留一个协程处理通道中剩余的数据，尽量保证工作进程的数据转发到用户
				r.done()
			}
			return
		}
	}
}

func (r *setUserLoginStatusBusiness) Send(userLoginStatus *setUserLoginStatus.SetUserLoginStatus) {
	select {
	case <-quit.Ctx.Done():
		return
	default:
		r.ch <- userLoginStatus
	}
}

var SetUserLoginStatus *setUserLoginStatusBusiness

func init() {
	SetUserLoginStatus = &setUserLoginStatusBusiness{ch: make(chan *setUserLoginStatus.SetUserLoginStatus, 100)}
	for i := 0; i < 10; i++ {
		quit.Wg.Add(1)
		go SetUserLoginStatus.consumer(i)
	}
}
