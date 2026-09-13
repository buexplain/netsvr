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

// TopicCustomerIdToUniqIdsListRet 获取某几个主题的customerId以及对应uniqId列表的结果，
// key是网关地址，value是该网关返回的响应
type TopicCustomerIdToUniqIdsListRet struct {
	Data map[string]*netsvrProtocol.TopicCustomerIdToUniqIdsListResp
}

// TopicCustomerIds 该主题下出现过的customerId，跨网关合并去重（顺序不保证）。
// 请求的主题不存在时返回 nil。
func (t *TopicCustomerIdToUniqIdsListRet) TopicCustomerIds(topic string) []string {
	seen := make(map[string]struct{})
	var ret []string
	for _, v := range t.Data {
		topicItem, ok := v.GetItems()[topic]
		if !ok {
			continue
		}
		for customerId := range topicItem.GetItems() {
			if _, ok := seen[customerId]; ok {
				continue
			}
			seen[customerId] = struct{}{}
			ret = append(ret, customerId)
		}
	}
	return ret
}

// CustomerUniqIds 合并所有网关中「该主题下该客户」的uniqId。
// 一个连接只属于一个网关，各网关的uniqId不会重复，因此直接合并即可。
// 注意：按协议，该列表是该客户在当前网关内的全部连接，不限于该主题。
// 请求的主题或客户不存在时返回 nil。
func (t *TopicCustomerIdToUniqIdsListRet) CustomerUniqIds(topic, customerId string) []string {
	var ret []string
	for _, v := range t.Data {
		topicItem, ok := v.GetItems()[topic]
		if !ok {
			continue
		}
		item, ok := topicItem.GetItems()[customerId]
		if !ok {
			continue
		}
		ret = append(ret, item.GetUniqIds()...)
	}
	return ret
}
