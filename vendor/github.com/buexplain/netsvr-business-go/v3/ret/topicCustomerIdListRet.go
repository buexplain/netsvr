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
	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
)

// TopicCustomerIdListRet 获取某几个主题的customerId的结果，key是网关地址，value是该网关返回的响应
type TopicCustomerIdListRet struct {
	Data map[string]*netsvrProtocol.TopicCustomerIdListResp
}

// TopicCustomerIds 合并所有网关中该主题包含的customerId并去重；同一个客户可能连接到多个网关。
// 请求的主题不存在时返回 nil。
func (t *TopicCustomerIdListRet) TopicCustomerIds(topic string) []string {
	seen := make(map[string]struct{})
	var ret []string
	for _, v := range t.Data {
		item, ok := v.GetItems()[topic]
		if !ok {
			continue
		}
		ret = dedupAppend(ret, seen, item.GetCustomerIds())
	}
	return ret
}

// CustomerIds 合并所有网关的结果，返回「主题 → customerId 列表」；每个主题的列表已跨网关去重。
// 请求的主题没找到时，结果中不会有该主题。
func (t *TopicCustomerIdListRet) CustomerIds() map[string][]string {
	seen := make(map[string]map[string]struct{})
	ret := make(map[string][]string)
	for _, v := range t.Data {
		for topic, item := range v.GetItems() {
			s, ok := seen[topic]
			if !ok {
				s = make(map[string]struct{}, len(item.GetCustomerIds()))
				seen[topic] = s
			}
			ret[topic] = dedupAppend(ret[topic], s, item.GetCustomerIds())
		}
	}
	return ret
}
