package session

import "sync"

type Info struct {
	//网关session id，初次设定后便是只读，所以可以导出
	sessionId uint32
	//用户信息
	user []byte
	//登录状态
	loginStatus int8
	mux         sync.RWMutex
}

func NewInfo(sessionId uint32) *Info {
	return &Info{
		sessionId:   sessionId,
		loginStatus: LoginStatusWait,
		mux:         sync.RWMutex{},
	}
}

func (r *Info) GetSessionId() uint32 {
	return r.sessionId
}

func (r *Info) GetUser() []byte {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.user
}

func (r *Info) SetUser(user []byte) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.user = user
}

func (r *Info) GetLoginStatus() int8 {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.loginStatus
}

func (r *Info) SetLoginStatus(loginStatus int8) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.loginStatus = loginStatus
}

func (r *Info) SetLoginStatusOk(userInfo []byte) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.loginStatus = LoginStatusOk
	r.user = userInfo
}

const (
	// LoginStatusWait 等待登录
	LoginStatusWait int8 = iota
	// LoginStatusIng 登录中LoginStatusIng
	LoginStatusIng
	// LoginStatusOk 登录成功
	LoginStatusOk
)
