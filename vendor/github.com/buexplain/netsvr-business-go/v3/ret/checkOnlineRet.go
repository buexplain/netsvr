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

// CheckOnlineRet 检查uniqId是否在线的结果，key是网关地址，value是该网关返回的响应
type CheckOnlineRet struct {
	Data map[string]*netsvrProtocol.CheckOnlineResp
}

// Has 判断某个uniqId是否在线，任意一个网关在线即为在线
func (c *CheckOnlineRet) Has(uniqId string) bool {
	for _, v := range c.Data {
		if slices.Contains(v.GetUniqIds(), uniqId) {
			return true
		}
	}
	return false
}

// UniqIds 合并所有网关的在线uniqId。
// 一个连接只属于一个网关，各网关的uniqId不会重复，因此直接合并即可。
func (c *CheckOnlineRet) UniqIds() []string {
	size := 0
	for _, v := range c.Data {
		size += len(v.GetUniqIds())
	}
	ret := make([]string, 0, size)
	for _, v := range c.Data {
		ret = append(ret, v.GetUniqIds()...)
	}
	return ret
}

// Len 在线连接数；各网关的uniqId不重复，直接累加即为总数，无需分配
func (c *CheckOnlineRet) Len() int {
	size := 0
	for _, v := range c.Data {
		size += len(v.GetUniqIds())
	}
	return size
}
