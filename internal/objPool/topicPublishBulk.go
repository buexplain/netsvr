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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"sync"
)

type topicPublishBulk struct {
	pool *sync.Pool
}

var TopicPublishBulk *topicPublishBulk

func (r *topicPublishBulk) Get() *netsvrProtocol.TopicPublishBulk {
	return r.pool.Get().(*netsvrProtocol.TopicPublishBulk)
}

func (r *topicPublishBulk) Put(topicPublishBulk *netsvrProtocol.TopicPublishBulk) {
	topicPublishBulk.Data = nil
	topicPublishBulk.Topics = nil
	r.pool.Put(topicPublishBulk)
}

func init() {
	TopicPublishBulk = &topicPublishBulk{
		pool: &sync.Pool{
			New: func() any {
				return &netsvrProtocol.TopicPublishBulk{}
			},
		},
	}
}
