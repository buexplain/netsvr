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

// CustomerIdListRet 获取网关全部customerId的结果，key是网关地址，value是该网关返回的响应
type CustomerIdListRet struct {
	Data map[string]*netsvrProtocol.CustomerIdListResp
}

// CustomerIds 合并所有网关的customerId并去重；同一个客户可能连接到多个网关
func (c *CustomerIdListRet) CustomerIds() []string {
	seen := make(map[string]struct{}, len(c.Data))
	var ret []string
	for _, v := range c.Data {
		ret = dedupAppend(ret, seen, v.GetCustomerIds())
	}
	return ret
}

// Has 判断某个customerId是否在线
func (c *CustomerIdListRet) Has(customerId string) bool {
	for _, v := range c.Data {
		if slices.Contains(v.GetCustomerIds(), customerId) {
			return true
		}
	}
	return false
}

// Len 去重后的在线客户数；同一个客户可能连接到多个网关，用 map 去重后取长度即可
func (c *CustomerIdListRet) Len() int {
	seen := make(map[string]struct{})
	for _, v := range c.Data {
		for _, customerId := range v.GetCustomerIds() {
			seen[customerId] = struct{}{}
		}
	}
	return len(seen)
}
