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

// ConnInfoRet 获取uniqId连接信息的结果，key是网关地址，value是该网关返回的响应
type ConnInfoRet struct {
	Data map[string]*netsvrProtocol.ConnInfoResp
}

// ToMap 合并所有网关的结果，key是uniqId，value是连接信息
func (c *ConnInfoRet) ToMap() map[string]*netsvrProtocol.ConnInfoRespItem {
	ret := make(map[string]*netsvrProtocol.ConnInfoRespItem)
	for _, v := range c.Data {
		for uniqId, item := range v.GetItems() {
			ret[uniqId] = item
		}
	}
	return ret
}

// Get 获取某个uniqId的连接信息；一个连接只属于一个网关，命中即返回
func (c *ConnInfoRet) Get(uniqId string) (*netsvrProtocol.ConnInfoRespItem, bool) {
	for _, v := range c.Data {
		if item, ok := v.GetItems()[uniqId]; ok {
			return item, true
		}
	}
	return nil, false
}
