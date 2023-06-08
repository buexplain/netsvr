/**
* Copyright 2023 buexplain@qq.com
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

// Package info 保持客户连接的session数据模块
package info

import (
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"sync"
)

type Info struct {
	//客户在网关中的唯一id
	uniqId string
	//客户存储在网关中的数据
	session string
	//客户订阅的主题
	topics map[string]struct{}
	//锁
	mux *sync.RWMutex
	//关闭信号
	closed chan struct{}
}

func NewInfo(uniqId string) *Info {
	return &Info{
		uniqId: uniqId,
		topics: nil,
		mux:    &sync.RWMutex{},
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

func (r *Info) SetUniqIdAndPUllTopics(uniqId string) (topics map[string]struct{}) {
	r.uniqId = uniqId
	ret := r.topics
	r.topics = nil
	return ret
}

func (r *Info) SetUniqIdAndGetTopics(uniqId string) (topics []string) {
	r.uniqId = uniqId
	if len(r.topics) == 0 {
		return nil
	}
	//复制一份主题数据出来
	topics = make([]string, 0, len(r.topics))
	for topic := range r.topics {
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

func (r *Info) GetToProtocolTransfer(tf *netsvrProtocol.Transfer) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	tf.Session = r.session
	tf.UniqId = r.uniqId
}

func (r *Info) GetToProtocolInfoResp(infoRespConnInfoRespItem *netsvrProtocol.ConnInfoRespItem) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	infoRespConnInfoRespItem.Session = r.session
	if len(r.topics) > 0 {
		infoRespConnInfoRespItem.Topics = make([]string, 0, len(r.topics))
		for topic := range r.topics {
			infoRespConnInfoRespItem.Topics = append(infoRespConnInfoRespItem.Topics, topic)
		}
	}
}

func (r *Info) Clear() (topics map[string]struct{}, uniqId string, session string) {
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

func (r *Info) PullTopics() map[string]struct{} {
	ret := r.topics
	r.topics = nil
	return ret
}

// SubscribeTopics 订阅，并返回当前的uniqId
func (r *Info) SubscribeTopics(topics []string) (currentUniqId string) {
	if r.topics == nil {
		r.topics = make(map[string]struct{}, len(topics))
	}
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		if _, ok := r.topics[topic]; !ok {
			r.topics[topic] = struct{}{}
		}
	}
	return r.uniqId
}

// UnsubscribeTopics 取消订阅，并返回当前的uniqId
func (r *Info) UnsubscribeTopics(topics []string) (currentUniqId string) {
	for _, topic := range topics {
		delete(r.topics, topic)
	}
	currentUniqId = r.uniqId
	return
}

// UnsubscribeTopic 取消订阅
func (r *Info) UnsubscribeTopic(topic string) bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	if _, ok := r.topics[topic]; ok {
		delete(r.topics, topic)
		return true
	}
	return false
}
