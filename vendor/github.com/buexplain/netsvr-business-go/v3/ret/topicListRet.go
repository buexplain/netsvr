/**
* Copyright 2024 buexplain@qq.com
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

package ret

import (
	"slices"

	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
)

// TopicListRet 获取网关全部主题的结果，key是网关地址，value是该网关返回的响应
type TopicListRet struct {
	Data map[string]*netsvrProtocol.TopicListResp
}

// Topics 合并所有网关的主题并去重；同名主题可能分布在多个网关
func (t *TopicListRet) Topics() []string {
	seen := make(map[string]struct{}, len(t.Data))
	var ret []string
	for _, v := range t.Data {
		ret = dedupAppend(ret, seen, v.GetTopics())
	}
	return ret
}

// Has 判断网关中是否存在该主题
func (t *TopicListRet) Has(topic string) bool {
	for _, v := range t.Data {
		if slices.Contains(v.GetTopics(), topic) {
			return true
		}
	}
	return false
}
