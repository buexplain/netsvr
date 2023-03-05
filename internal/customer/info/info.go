// Package info 保持客户连接的session数据模块
package info

import (
	"netsvr/internal/protocol"
	"sync"
)

type Info struct {
	uniqId  string
	session string
	topics  []string
	mux     sync.RWMutex
	closed  chan struct{}
}

func NewInfo(uniqId string) *Info {
	return &Info{
		uniqId: uniqId,
		topics: nil,
		mux:    sync.RWMutex{},
		closed: make(chan struct{}),
	}
}

func (r *Info) Close() {
	select {
	case <-r.closed:
		return
	default:
		close(r.closed)
	}
}

func (r *Info) IsClosed() bool {
	select {
	case <-r.closed:
		return true
	default:
		return false
	}
}

func (r *Info) MuxLock() {
	r.mux.Lock()
}

func (r *Info) MuxUnLock() {
	r.mux.Unlock()
}

func (r *Info) GetUniqId() string {
	return r.uniqId
}

func (r *Info) SetUniqIdAndPUllTopics(uniqId string) (topics []string) {
	r.uniqId = uniqId
	if len(r.topics) == 0 {
		return nil
	}
	ret := r.topics
	r.topics = nil
	return ret
}

func (r *Info) SetUniqIdAndGetTopics(uniqId string) (topics []string) {
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

func (r *Info) GetSession() string {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.session
}

func (r *Info) SetSession(session string) {
	r.session = session
}

func (r *Info) GetToProtocolTransfer(tf *protocol.Transfer) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	tf.Session = r.session
	tf.UniqId = r.uniqId
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

func (r *Info) Clear() (topics []string, uniqId string, session string) {
	//删除所有订阅
	topics = r.topics
	r.topics = nil
	//删除唯一id
	uniqId = r.uniqId
	r.uniqId = ""
	//删除session
	session = r.session
	r.session = ""
	return
}

func (r *Info) PullTopics() []string {
	if len(r.topics) == 0 {
		return nil
	}
	ret := r.topics
	r.topics = nil
	return ret
}

// SubscribeTopics 订阅，并返回成功订阅的主题，以及当前的uniqId
func (r *Info) SubscribeTopics(topics []string, lock bool) (realSubscribeTopics []string, currentUniqId string) {
	if len(topics) == 0 {
		//这里返回空的uniqId，因为无妨
		return nil, ""
	}
	if lock {
		r.mux.Lock()
		defer r.mux.Unlock()
	}
	//如果是个空，则全部赋值
	if r.topics == nil {
		r.topics = make([]string, 0, len(topics))
		for _, topic := range topics {
			r.topics = append(r.topics, topic)
		}
		return topics, r.uniqId
	}
	//不为空，判断是否存在，不存在的，则赋值
	realSubscribeTopics = make([]string, 0)
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
			realSubscribeTopics = append(realSubscribeTopics, topic)
		}
	}
	currentUniqId = r.uniqId
	return
}

// UnsubscribeTopics 取消订阅，并返回成功取消订阅的主题，以及当前的uniqId
func (r *Info) UnsubscribeTopics(topics []string) (realUnsubscribeTopics []string, currentUniqId string) {
	if len(topics) == 0 {
		return nil, ""
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	//为空，则无需任何操作
	if r.topics == nil {
		return nil, ""
	}
	//判断已经存在的才删除
	realUnsubscribeTopics = make([]string, 0)
	for _, topic := range topics {
		for k, has := range r.topics {
			if topic == has {
				r.topics = append(r.topics[0:k], r.topics[k+1:]...)
				realUnsubscribeTopics = append(realUnsubscribeTopics, topic)
				break
			}
		}
	}
	currentUniqId = r.uniqId
	return
}

// UnsubscribeTopic 取消订阅
func (r *Info) UnsubscribeTopic(topic string) bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	//为空，则无需任何操作
	if r.topics == nil {
		return false
	}
	//判断已经存在的才删除
	for k, has := range r.topics {
		if topic == has {
			r.topics = append(r.topics[0:k], r.topics[k+1:]...)
			return true
		}
	}
	return false
}
