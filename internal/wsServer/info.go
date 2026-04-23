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

package wsServer

import (
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"netsvr/configs"
	"netsvr/internal/wsServer/uniqIdGen"
	"sync"
	"sync/atomic"
	"time"
)

// info 保持客户连接的信息的结构体
type info struct {
	//锁（放在最前面，确保8字节对齐）
	rwMutex sync.RWMutex
	//在进程内的唯一id
	id uint64
	//当前窗口的开始时间（Unix时间戳，秒）
	limitWindowStart int64
	//客户在网关中的唯一id
	uniqId string
	//客户在业务系统中的唯一id
	customerId string
	//客户存储在网关中的数据
	session string
	//客户订阅的主题
	topics map[string]struct{}
	//当前窗口内的请求数
	limitWindowRequests int32
	//最后活跃时间
	lastActiveTime uint32 //2106-02-07 14:28:15之后，本程序不会支持
}

func newInfo() info {
	uniqId, id := uniqIdGen.New()
	return info{
		uniqId:              uniqId,
		id:                  id,
		limitWindowStart:    0, // 初始化为0，第一次调用Allow时会设置
		limitWindowRequests: 0 - configs.Config.Customer.LimitZeroWindowMaxRequests,
		topics:              nil,
		rwMutex:             sync.RWMutex{},
		lastActiveTime:      uint32(time.Now().Unix()),
	}
}

func (r *info) UpdateLastActiveTimeOnSafe() {
	atomic.StoreUint32(&r.lastActiveTime, uint32(time.Now().Unix()))
}

func (r *info) GetLastActiveTimeOnSafe() uint32 {
	return atomic.LoadUint32(&r.lastActiveTime)
}

func (r *info) RLock() {
	r.rwMutex.RLock()
}

func (r *info) RUnlock() {
	r.rwMutex.RUnlock()
}

func (r *info) Lock() {
	r.rwMutex.Lock()
}

func (r *info) UnLock() {
	r.rwMutex.Unlock()
}

func (r *info) Allow() bool {
	//先消耗初始的固定请求数
	if r.limitWindowRequests < 0 {
		r.limitWindowRequests++
		return true
	}
	//检查是否进入下一个窗口
	now := time.Now().Unix()
	if now-r.limitWindowStart > configs.Config.Customer.LimitWindowSizeSeconds {
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

func (r *info) Snapshot() (uniqId string, customerId string, session string, topics []string) {
	topics = r.getTopics()
	uniqId = r.uniqId
	session = r.session
	customerId = r.customerId
	return
}

func (r *info) GetUniqIdOnSafe() string {
	//这里无需锁，因为没有地方会修改uniqId
	return r.uniqId
}
func (r *info) GetIdOnSafe() uint64 {
	//这里无需锁，因为没有地方会修改uniqId
	return r.id
}

func (r *info) GetCustomerIdOnSafe() string {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	return r.customerId
}

func (r *info) IsLoginOnSafe() bool {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	//有session或customerId，则视为登录状态的连接
	return r.session != "" || r.customerId != ""
}

func (r *info) GetConnInfoOnSafe(connInfoReq *netsvrProtocol.ConnInfoReq, connInfoRespItem *netsvrProtocol.ConnInfoRespItem) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	if connInfoReq.ReqSession {
		connInfoRespItem.Session = r.session
	}
	if connInfoReq.ReqCustomerId {
		connInfoRespItem.CustomerId = r.customerId
	}
	if connInfoReq.ReqTopic && len(r.topics) > 0 {
		connInfoRespItem.Topics = r.getTopics()
	}
}

func (r *info) GetConnInfoByCustomerIdOnSafe(connInfoReq *netsvrProtocol.ConnInfoByCustomerIdReq, connInfoRespItem *netsvrProtocol.ConnInfoByCustomerIdRespItem) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()
	if connInfoReq.ReqSession {
		connInfoRespItem.Session = r.session
	}
	if connInfoReq.ReqUniqId {
		connInfoRespItem.UniqId = r.uniqId
	}
	if connInfoReq.ReqTopic && len(r.topics) > 0 {
		connInfoRespItem.Topics = r.getTopics()
	}
}

func (r *info) GetCustomerId() string {
	return r.customerId
}

func (r *info) getTopics() []string {
	topics := make([]string, 0, len(r.topics))
	for topic := range r.topics {
		topics = append(topics, topic)
	}
	return topics
}

func (r *info) SetCustomerId(customerId string) {
	r.customerId = customerId
}

func (r *info) SetSession(session string) {
	r.session = session
}

func (r *info) PullTopics() map[string]struct{} {
	ret := r.topics
	r.topics = nil
	return ret
}

// SubscribeTopics 订阅，并返回当前的uniqId
func (r *info) SubscribeTopics(topics []string) {
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
func (r *info) UnsubscribeTopics(topics []string) {
	for _, topic := range topics {
		delete(r.topics, topic)
	}
}

// UnsubscribeTopicOnSafe 取消订阅
func (r *info) UnsubscribeTopicOnSafe(topic string) bool {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()
	if _, ok := r.topics[topic]; ok {
		delete(r.topics, topic)
		return true
	}
	return false
}
