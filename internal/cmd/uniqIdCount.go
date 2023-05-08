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

package cmd

import (
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	customerManager "netsvr/internal/customer/manager"
	workerManager "netsvr/internal/worker/manager"
)

// UniqIdCount 获取网关中uniqId的数量
func UniqIdCount(_ []byte, processor *workerManager.ConnProcessor) {
	ret := &netsvrProtocol.UniqIdCountResp{}
	ret.ServerId = int32(configs.Config.ServerId)
	ret.Count = int32(customerManager.Manager.Len())
	route := &netsvrProtocol.Router{}
	route.Cmd = netsvrProtocol.Cmd_UniqIdCount
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
