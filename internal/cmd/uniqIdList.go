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
	"netsvr/configs"
	customerManager "netsvr/internal/customer/manager"
	workerManager "netsvr/internal/worker/manager"
)

// UniqIdList 获取网关中全部的uniqId
func UniqIdList(_ []byte, processor *workerManager.ConnProcessor) {
	//加1024的目的是担心获取连接的过程中又有连接进来，导致slice的底层发生扩容，引起内存拷贝
	uniqIds := make([]string, 0, customerManager.Manager.Len()+1024)
	for _, c := range customerManager.Manager {
		c.GetUniqIds(&uniqIds)
	}
	ret := &netsvrProtocol.UniqIdListResp{}
	ret.ServerId = int32(configs.Config.ServerId)
	ret.UniqIds = uniqIds
	processor.Send(ret, netsvrProtocol.Cmd_UniqIdList)
}
