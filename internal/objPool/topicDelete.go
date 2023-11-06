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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v2/netsvr"
	"sync"
)

type topicDelete struct {
	pool *sync.Pool
}

var TopicDelete *topicDelete

func (r *topicDelete) Get() *netsvrProtocol.TopicDelete {
	return r.pool.Get().(*netsvrProtocol.TopicDelete)
}

func (r *topicDelete) Put(topicDelete *netsvrProtocol.TopicDelete) {
	topicDelete.Data = nil
	topicDelete.Topics = nil
	r.pool.Put(topicDelete)
}

func init() {
	TopicDelete = &topicDelete{
		pool: &sync.Pool{
			New: func() any {
				return &netsvrProtocol.TopicDelete{}
			},
		},
	}
}
