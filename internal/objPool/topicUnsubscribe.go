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

package objPool

import (
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"sync"
)

type topicUnsubscribe struct {
	pool *sync.Pool
}

var TopicUnsubscribe *topicUnsubscribe

func (r *topicUnsubscribe) Get() *netsvrProtocol.TopicUnsubscribe {
	return r.pool.Get().(*netsvrProtocol.TopicUnsubscribe)
}

func (r *topicUnsubscribe) Put(topicUnsubscribe *netsvrProtocol.TopicUnsubscribe) {
	topicUnsubscribe.Data = nil
	topicUnsubscribe.Topics = nil
	topicUnsubscribe.UniqId = ""
	r.pool.Put(topicUnsubscribe)
}

func init() {
	TopicUnsubscribe = &topicUnsubscribe{
		pool: &sync.Pool{
			New: func() any {
				return &netsvrProtocol.TopicUnsubscribe{}
			},
		},
	}
}
