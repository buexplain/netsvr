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

// ConnInfoByCustomerIdRet 获取customerId连接信息的结果，key是网关地址，value是该网关返回的响应
type ConnInfoByCustomerIdRet struct {
	Data map[string]*netsvrProtocol.ConnInfoByCustomerIdResp
}

// ToMap 合并所有网关的结果，返回「customerId → 该客户的全部连接」；同一个客户可能连接到多个网关
func (c *ConnInfoByCustomerIdRet) ToMap() map[string][]*netsvrProtocol.ConnInfoByCustomerIdRespItem {
	ret := make(map[string][]*netsvrProtocol.ConnInfoByCustomerIdRespItem)
	for _, v := range c.Data {
		for customerId, items := range v.GetItems() {
			ret[customerId] = append(ret[customerId], items.GetItems()...)
		}
	}
	return ret
}

// Get 获取某个customerId的全部连接；同一个客户可能连接到多个网关，各网关的连接会合并
func (c *ConnInfoByCustomerIdRet) Get(customerId string) []*netsvrProtocol.ConnInfoByCustomerIdRespItem {
	var ret []*netsvrProtocol.ConnInfoByCustomerIdRespItem
	for _, v := range c.Data {
		if items, ok := v.GetItems()[customerId]; ok {
			ret = append(ret, items.GetItems()...)
		}
	}
	return ret
}
