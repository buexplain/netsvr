package session

import (
	"netsvr/internal/protocol/toWorker/respSessionInfo"
	"sync"
)

type Info struct {
	//网关session id，初次设定后便是只读，所以可以导出
	sessionId uint32
	//用户信息
	user string
	//登录状态
	loginStatus int8
	//当前连接订阅的主题
	topics []string
	//当前连接最后发送消息的时间
	lastActiveTime int64
	mux            sync.RWMutex
}

func NewInfo(sessionId uint32) *Info {
	return &Info{
		sessionId:      sessionId,
		loginStatus:    LoginStatusWait,
		topics:         []string{},
		lastActiveTime: 0,
		mux:            sync.RWMutex{},
	}
}

// SetLastActiveTime 更新连接的最后活跃时间
func (r *Info) SetLastActiveTime(lastActiveTime int64) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.lastActiveTime = lastActiveTime
}

func (r *Info) GetLastActiveTime() int64 {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.lastActiveTime
}

func (r *Info) GetSessionId() uint32 {
	return r.sessionId
}

func (r *Info) GetToRespSessionInfo(respSessionInfo *respSessionInfo.RespSessionInfo) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	respSessionInfo.User = r.user
	if r.loginStatus == LoginStatusOk {
		respSessionInfo.LoginStatus = true
	} else {
		respSessionInfo.LoginStatus = false
	}
	respSessionInfo.Topics = make([]string, 0, len(r.topics))
	respSessionInfo.Topics = append(respSessionInfo.Topics, r.topics...)
}

func (r *Info) GetUser() string {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.user
}

func (r *Info) SetUser(user string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.user = user
}

func (r *Info) SetUserOnLoginStatusOk(user string) bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.loginStatus == LoginStatusOk {
		r.user = user
		return true
	}
	return false
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
	r.user = ""
}

func (r *Info) SetLoginStatusOk(userInfo string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.loginStatus = LoginStatusOk
	r.user = userInfo
}

func (r *Info) GetTopics() []string {
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := make([]string, 0, len(r.topics))
	ret = append(ret, r.topics...)
	return ret
}

func (r *Info) Subscribe(topics []string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		ok := false
		for _, has := range r.topics {
			if topic == has {
				ok = true
				break
			}
		}
		if !ok {
			r.topics = append(r.topics, topic)
		}
	}
}

func (r *Info) Unsubscribe(topics []string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, topic := range topics {
		for k, has := range r.topics {
			if topic == has {
				r.topics = append(r.topics[0:k], r.topics[k+1:]...)
				break
			}
		}
	}
}

const (
	// LoginStatusWait 等待登录
	LoginStatusWait int8 = iota
	// LoginStatusIng 登录中LoginStatusIng
	LoginStatusIng
	// LoginStatusOk 登录成功
	LoginStatusOk
)
