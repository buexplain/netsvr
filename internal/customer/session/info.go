package session

import (
	"netsvr/internal/protocol"
	"sync"
)

const (
	// LoginStatusWait 等待登录
	LoginStatusWait int8 = iota
	// LoginStatusIng 登录中LoginStatusIng
	LoginStatusIng
	// LoginStatusOk 登录成功
	LoginStatusOk
)

type Info struct {
	//网关session id，初次设定后便是只读
	sessionId uint32
	//客户信息
	userInfo string
	//客户在业务系统中的唯一id
	userId string
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

func (r *Info) GetToSessionInfoObj(sessionInfo *protocol.SessionInfoResp) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	sessionInfo.UserInfo = r.userInfo
	sessionInfo.UserId = r.userId
	if r.loginStatus == LoginStatusOk {
		sessionInfo.LoginStatus = true
	} else {
		sessionInfo.LoginStatus = false
	}
	sessionInfo.Topics = make([]string, 0, len(r.topics))
	sessionInfo.Topics = append(sessionInfo.Topics, r.topics...)
}

func (r *Info) GetToConnCloseObj(connClose *protocol.ConnClose) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	connClose.SessionId = r.sessionId
	connClose.UserInfo = r.userInfo
	connClose.UserId = r.userId
}

func (r *Info) GetToTransferObj(transfer *protocol.Transfer) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	transfer.SessionId = r.sessionId
	transfer.UserInfo = r.userInfo
	transfer.UserId = r.userId
}

func (r *Info) UpUserInfoOnLoginStatusOk(userInfo string, userId string) bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.loginStatus == LoginStatusOk && r.userId == userId {
		r.userInfo = userInfo
		return true
	}
	return false
}

func (r *Info) GetLoginStatus() int8 {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.loginStatus
}

func (r *Info) SetLoginStatusWait() {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.loginStatus = LoginStatusWait
	r.userInfo = ""
	r.userId = ""
}

func (r *Info) SetLoginStatusIng() {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.loginStatus = LoginStatusIng
	r.userInfo = ""
	r.userId = ""
}

func (r *Info) SetLoginStatusOk(userInfo string, userId string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.loginStatus = LoginStatusOk
	r.userInfo = userInfo
	r.userId = userId
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
