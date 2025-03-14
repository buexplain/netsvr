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
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"netsvr/configs"
	"sync"
	"time"
)

// Info 保持客户连接的信息的结构体
type Info struct {
	//客户在网关中的唯一id
	uniqId string
	//客户在业务系统中的唯一id
	customerId string
	//客户存储在网关中的数据
	session string
	//当前窗口的开始时间
	limitWindowStart time.Time
	//客户订阅的主题
	topics map[string]struct{}
	//锁
	rwMutex *sync.RWMutex
	//当前窗口内的请求数
	limitWindowRequests uint32
}

var zeroTime = time.Time{}

func NewInfo(uniqId string) *Info {
	return &Info{
		uniqId:           uniqId,
		limitWindowStart: zeroTime,
		topics:           nil,
		rwMutex:          &sync.RWMutex{},
	}
}

func (r *Info) Lock() {
	r.rwMutex.Lock()
}

func (r *Info) UnLock() {
	r.rwMutex.Unlock()
}

func (r *Info) Allow() bool {
	//先消耗初始的固定请求数
	if r.limitWindowStart == zeroTime {
		if r.limitWindowRequests < configs.Config.Customer.LimitZeroWindowMaxRequests {
			r.limitWindowRequests++
			return true
		}
		r.limitWindowStart = time.Now()
		r.limitWindowRequests = 0 //固定桶内数量消耗完毕，继续下放固定窗口内的请求数，允许固定数量限制与固定窗口数量限制之间平滑过渡
		return false
	}
	//检查是否进入下一个窗口
	now := time.Now()
	if now.Sub(r.limitWindowStart) >= configs.Config.Customer.LimitWindowSize {
		r.limitWindowStart = now
		r.limitWindowRequests = 0
	}
	//检查当前窗口内的请求数是否超过限制
	if r.limitWindowRequests < configs.Config.Customer.LimitWindowMaxRequests {
		r.limitWindowRequests++
		return true
	}
	return false
}

func (r *Info) GetCustomerId() string {
	return r.customerId
}

func (r *Info) GetUniqId() string {
	return r.uniqId
}

func (r *Info) GetSession() string {
	return r.session
}

func (r *Info) GetTopics() []string {
	topics := make([]string, 0, len(r.topics))
	for topic := range r.topics {
		topics = append(topics, topic)
	}
	return topics
}

func (r *Info) IsLogin() bool {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	//有session或customerId，则视为登录状态的连接
	return r.session != "" || r.customerId != ""
}

func (r *Info) SetSession(session string) {
	r.session = session
}

func (r *Info) SetCustomerId(customerId string) {
	r.customerId = customerId
}

func (r *Info) GetConnInfoOnSafe(connInfoReq *netsvrProtocol.ConnInfoReq, connInfoRespItem *netsvrProtocol.ConnInfoRespItem) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	if connInfoReq.ReqSession {
		connInfoRespItem.Session = r.session
	}
	if connInfoReq.ReqCustomerId {
		connInfoRespItem.CustomerId = r.customerId
	}
	if connInfoReq.ReqTopic && len(r.topics) > 0 {
		connInfoRespItem.Topics = r.GetTopics()
	}
}

func (r *Info) GetConnInfoByCustomerIdOnSafe(connInfoReq *netsvrProtocol.ConnInfoByCustomerIdReq, connInfoRespItem *netsvrProtocol.ConnInfoByCustomerIdRespItem) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	if connInfoReq.ReqSession {
		connInfoRespItem.Session = r.session
	}
	if connInfoReq.ReqUniqId {
		connInfoRespItem.UniqId = r.uniqId
	}
	if connInfoReq.ReqTopic && len(r.topics) > 0 {
		connInfoRespItem.Topics = r.GetTopics()
	}
}

// Clear 清空session
func (r *Info) Clear() (uniqId string, customerId string, session string, topics []string) {
	topics = r.GetTopics()
	r.topics = nil
	//删除唯一id
	uniqId = r.uniqId
	r.uniqId = ""
	//删除session
	session = r.session
	r.session = ""
	//删除客户的业务id
	customerId = r.customerId
	r.customerId = ""
	return
}

func (r *Info) PullTopics() map[string]struct{} {
	ret := r.topics
	r.topics = nil
	return ret
}

// SubscribeTopics 订阅，并返回当前的uniqId
func (r *Info) SubscribeTopics(topics []string) {
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
}

// UnsubscribeTopics 取消订阅，并返回当前的uniqId
func (r *Info) UnsubscribeTopics(topics []string) {
	for _, topic := range topics {
		delete(r.topics, topic)
	}
}

// UnsubscribeTopic 取消订阅
func (r *Info) UnsubscribeTopic(topic string) bool {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()
	if _, ok := r.topics[topic]; ok {
		delete(r.topics, topic)
		return true
	}
	return false
}
