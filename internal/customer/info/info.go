package info

import (
	"github.com/antlabs/timer"
	"netsvr/internal/protocol"
	"sync"
)

type Info struct {
	uniqId         string
	session        string
	topics         []string
	lastActiveTime int64
	HeartbeatNode  timer.TimeNoder
	mux            sync.RWMutex
}

func NewInfo(uniqId string) *Info {
	return &Info{
		uniqId:         uniqId,
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

func (r *Info) GetUniqId() string {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.uniqId
}

func (r *Info) SetUniqIdAndGetTopics(uniqId string) (topics []string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.uniqId = uniqId
	if len(r.topics) == 0 {
		return nil
	}
	topics = make([]string, 0, len(r.topics))
	for _, topic := range r.topics {
		topics = append(topics, topic)
	}
	return topics
}

func (r *Info) SetSession(session string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.session = session
}

func (r *Info) GetToProtocolTransfer(tf *protocol.Transfer) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	tf.Session = r.session
	tf.UniqId = r.uniqId
}

func (r *Info) GetToProtocolConnClose(cl *protocol.ConnClose) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	cl.Session = r.session
	cl.UniqId = r.uniqId
}

func (r *Info) GetToProtocolInfoResp(infoResp *protocol.InfoResp) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	infoResp.UniqId = r.uniqId
	infoResp.Session = r.session
	if len(r.topics) > 0 {
		infoResp.Topics = make([]string, 0, len(r.topics))
		for _, topic := range r.topics {
			infoResp.Topics = append(infoResp.Topics, topic)
		}
	}
}

func (r *Info) PullTopics() []string {
	if len(r.topics) == 0 {
		return nil
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := r.topics
	r.topics = []string{}
	return ret
}

// Subscribe 订阅，并返回成功订阅的主题
func (r *Info) Subscribe(topics []string) []string {
	if len(topics) == 0 {
		return nil
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := make([]string, 0)
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
			ret = append(ret, topic)
		}
	}
	return ret
}

// Unsubscribe 取消订阅，并返回成功取消订阅的主题
func (r *Info) Unsubscribe(topics []string) []string {
	if len(topics) == 0 {
		return nil
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := make([]string, 0)
	for _, topic := range topics {
		for k, has := range r.topics {
			if topic == has {
				r.topics = append(r.topics[0:k], r.topics[k+1:]...)
				ret = append(ret, topic)
				break
			}
		}
	}
	return ret
}
