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

// TopicUniqIdListRet 获取某几个主题包含的uniqId的结果，key是网关地址，value是该网关返回的响应
type TopicUniqIdListRet struct {
	Data map[string]*netsvrProtocol.TopicUniqIdListResp
}

// TopicUniqIds 合并所有网关中该主题包含的uniqId。
// 一个连接只属于一个网关，各网关的uniqId不会重复，因此直接合并即可。
// 请求的主题不存在时返回 nil。
func (t *TopicUniqIdListRet) TopicUniqIds(topic string) []string {
	var ret []string
	for _, v := range t.Data {
		item, ok := v.GetItems()[topic]
		if !ok {
			continue
		}
		ret = append(ret, item.GetUniqIds()...)
	}
	return ret
}

// UniqIds 合并所有网关的结果，返回「主题 → uniqId 列表」。
// 同一个主题的uniqId分布在不同网关且不会重复，因此按键分组后直接合并即可。
// 请求的主题没找到时，结果中不会有该主题。
func (t *TopicUniqIdListRet) UniqIds() map[string][]string {
	ret := make(map[string][]string)
	for _, v := range t.Data {
		for topic, item := range v.GetItems() {
			ret[topic] = append(ret[topic], item.GetUniqIds()...)
		}
	}
	return ret
}
