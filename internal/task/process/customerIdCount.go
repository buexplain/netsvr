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

package process

import (
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"net"
	"netsvr/internal/customer/binder"
)

// customerIdCount 获取customerId的数量
func customerIdCount(_ []byte, taskConn net.Conn) {
	ret := &netsvrProtocol.CustomerIdCountResp{}
	ret.Count = int32(binder.Binder.Len())
	send(taskConn, ret, netsvrProtocol.Cmd_CustomerIdCount)
}
