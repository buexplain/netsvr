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

// 本文件是「客户 <-> 业务进程」这层报文的结构定义：
// 发出去的命令、收回来的消息，以及各命令回包中最内层 data 的形状。
package integration

import (
	"encoding/json"
	"testing"

	"netsvr/test/pkg/protocol"
)

// ============================== 收发报文 ==============================

// clientCmd 客户发给业务进程的命令，形状与 test/pkg/protocol.ClientRouter 一致
type clientCmd struct {
	Cmd  protocol.Cmd `json:"cmd"`
	Data string       `json:"data"`
}

// respMessage 业务进程推给客户的一条消息：{"cmd":N,"data":{...}}
type respMessage struct {
	Cmd  protocol.Cmd    `json:"cmd"`
	Data json.RawMessage `json:"data"`
}

// envelope 业务进程回包的 data 层，统一是 {code,message,data}
type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// into 把回包里最内层的 data 解析到 v；data 为空时保持 v 的零值
func (e *envelope) into(t *testing.T, v any) {
	t.Helper()
	if len(e.Data) == 0 {
		return
	}
	if err := json.Unmarshal(e.Data, v); err != nil {
		t.Fatalf("解析回包 data 失败：%v（原始：%s）", err, e.Data)
	}
}

// ============================== 回包 data 的形状 ==============================
//
// 形状由 test/business/internal/cmd 下各处理器决定：多数命令直接把 SDK 查询结果
// （map[网关地址]响应）当作 data 下发，少数命令会再包一层
// connInfo / list / topics / count / customerIds。
// 这里只声明断言需要的字段，其余字段由 encoding/json 忽略。

// connOpenPayload 连接打开回包
type connOpenPayload struct {
	UniqId string `json:"uniqId"`
}

// clientInfoPayload 登录/伪造登录回包里的客户信息
type clientInfoPayload struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	UniqId string `json:"uniqId"`
}

// fromUserMessage 单播/广播/发布等投递给客户的内容
type fromUserMessage struct {
	FromUser string `json:"fromUser"`
	Message  string `json:"message"`
}

// uniqIdListPayload 对应 RouterUniqIdList / RouterCheckOnline：{网关地址: {uniqIds: [...]}}
type uniqIdListPayload map[string]struct {
	UniqIds []string `json:"uniqIds"`
}

func (p uniqIdListPayload) uniqIds() []string {
	var ret []string
	for _, v := range p {
		ret = append(ret, v.UniqIds...)
	}
	return ret
}

// countPayload 对应 RouterUniqIdCount / RouterTopicCount：{count: N}
type countPayload struct {
	Count int32 `json:"count"`
}

// connInfoItem 一条连接的详细信息
type connInfoItem struct {
	UniqId     string   `json:"uniqId"`
	CustomerId string   `json:"customerId"`
	Session    string   `json:"session"`
	Topics     []string `json:"topics"`
}

// connInfoPayload 对应 RouterConnInfo：{connInfo: {网关地址: {items: {uniqId: {...}}}}}
type connInfoPayload struct {
	ConnInfo map[string]struct {
		Items map[string]connInfoItem `json:"items"`
	} `json:"connInfo"`
}

func (p *connInfoPayload) items() map[string]connInfoItem {
	ret := make(map[string]connInfoItem)
	for _, v := range p.ConnInfo {
		for uniqId, item := range v.Items {
			ret[uniqId] = item
		}
	}
	return ret
}

// connInfoByCustomerIdPayload 对应 RouterConnInfoByCustomerId：
// {connInfo: {网关地址: {items: {customerId: {items: [连接信息...]}}}}}
type connInfoByCustomerIdPayload struct {
	ConnInfo map[string]struct {
		Items map[string]struct {
			Items []connInfoItem `json:"items"`
		} `json:"items"`
	} `json:"connInfo"`
}

func (p *connInfoByCustomerIdPayload) customerUniqIds(customerId string) []string {
	var ret []string
	for _, v := range p.ConnInfo {
		byCustomer, ok := v.Items[customerId]
		if !ok {
			continue
		}
		for _, item := range byCustomer.Items {
			ret = append(ret, item.UniqId)
		}
	}
	return ret
}

// topicListPayload 对应 RouterTopicList：{topics: {网关地址: {topics: [...]}}}
type topicListPayload struct {
	Topics map[string]struct {
		Topics []string `json:"topics"`
	} `json:"topics"`
}

func (p *topicListPayload) all() []string {
	var ret []string
	for _, v := range p.Topics {
		ret = append(ret, v.Topics...)
	}
	return ret
}

// topicUniqIdListPayload 对应 RouterTopicUniqIdList：{网关地址: {items: {topic: {uniqIds: [...]}}}}
type topicUniqIdListPayload map[string]struct {
	Items map[string]struct {
		UniqIds []string `json:"uniqIds"`
	} `json:"items"`
}

func (p topicUniqIdListPayload) topicUniqIds(topic string) []string {
	var ret []string
	for _, v := range p {
		if item, ok := v.Items[topic]; ok {
			ret = append(ret, item.UniqIds...)
		}
	}
	return ret
}

// topicUniqIdCountPayload 对应 RouterTopicUniqIdCount：{网关地址: {items: {topic: N}}}
type topicUniqIdCountPayload map[string]struct {
	Items map[string]int32 `json:"items"`
}

func (p topicUniqIdCountPayload) topicCount(topic string) int32 {
	var ret int32
	for _, v := range p {
		ret += v.Items[topic]
	}
	return ret
}

// topicCustomerIdListPayload 对应 RouterTopicCustomerIdList：
// {list: {网关地址: {items: {topic: {customerIds: [...]}}}}}
type topicCustomerIdListPayload struct {
	List map[string]struct {
		Items map[string]struct {
			CustomerIds []string `json:"customerIds"`
		} `json:"items"`
	} `json:"list"`
}

func (p *topicCustomerIdListPayload) topicCustomerIds(topic string) []string {
	var ret []string
	for _, v := range p.List {
		if item, ok := v.Items[topic]; ok {
			ret = append(ret, item.CustomerIds...)
		}
	}
	return ret
}

// topicCustomerIdCountPayload 对应 RouterTopicCustomerIdCount：
// {list: {网关地址: {items: {topic: N}}}}
type topicCustomerIdCountPayload struct {
	List map[string]struct {
		Items map[string]int32 `json:"items"`
	} `json:"list"`
}

func (p *topicCustomerIdCountPayload) topicCount(topic string) int32 {
	var ret int32
	for _, v := range p.List {
		ret += v.Items[topic]
	}
	return ret
}

// topicCustomerIdToUniqIdsListPayload 对应 RouterTopicCustomerIdToUniqIdsList：
// {list: {网关地址: {items: {topic: {items: {customerId: {uniqIds: [...]}}}}}}}
type topicCustomerIdToUniqIdsListPayload struct {
	List map[string]struct {
		Items map[string]struct {
			Items map[string]struct {
				UniqIds []string `json:"uniqIds"`
			} `json:"items"`
		} `json:"items"`
	} `json:"list"`
}

func (p *topicCustomerIdToUniqIdsListPayload) customerUniqIds(topic, customerId string) []string {
	var ret []string
	for _, v := range p.List {
		byTopic, ok := v.Items[topic]
		if !ok {
			continue
		}
		if item, ok := byTopic.Items[customerId]; ok {
			ret = append(ret, item.UniqIds...)
		}
	}
	return ret
}

// limitPayload 对应 RouterLimit：{网关地址: {onOpen: N, onMessage: N}}
type limitPayload map[string]struct {
	OnOpen    int32 `json:"onOpen"`
	OnMessage int32 `json:"onMessage"`
}

// customerIdListPayload 对应 RouterCustomerIdList：{customerIds: {网关地址: {customerIds: [...]}}}
type customerIdListPayload struct {
	CustomerIds map[string]struct {
		CustomerIds []string `json:"customerIds"`
	} `json:"customerIds"`
}

func (p *customerIdListPayload) all() []string {
	var ret []string
	for _, v := range p.CustomerIds {
		ret = append(ret, v.CustomerIds...)
	}
	return ret
}

// customerIdCountPayload 对应 RouterCustomerIdCount：{count: {网关地址: {count: N}}}
type customerIdCountPayload struct {
	Count map[string]struct {
		Count int32 `json:"count"`
	} `json:"count"`
}

func (p *customerIdCountPayload) total() int32 {
	var ret int32
	for _, v := range p.Count {
		ret += v.Count
	}
	return ret
}

// metricsPayload 业务进程把各网关的 metrics 拍平成一个数组后返回
type metricsPayload []struct {
	Description string  `json:"description"`
	Count       int64   `json:"count"`
	MeanRate    float32 `json:"meanRate"`
	Rate1       float32 `json:"rate1"`
	Rate5       float32 `json:"rate5"`
	Rate15      float32 `json:"rate15"`
}
