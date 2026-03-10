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
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
)

type ConnInfoRet struct {
	Data map[string]*netsvrProtocol.ConnInfoResp
}

func (c *ConnInfoRet) ToMap() map[string]*netsvrProtocol.ConnInfoRespItem {
	ret := make(map[string]*netsvrProtocol.ConnInfoRespItem, len(c.Data))
	for _, v := range c.Data {
		for uniqId, item := range v.Items {
			ret[uniqId] = item
		}
	}
	return ret
}
