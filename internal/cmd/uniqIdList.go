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
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// UniqIdList 获取网关中全部的uniqId
func UniqIdList(param []byte, processor *workerManager.ConnProcessor) {
	payload := netsvrProtocol.UniqIdListReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.UniqIdListReq failed")
		return
	}
	uniqIds := make([]string, 0, customerManager.Manager.Len())
	for _, c := range customerManager.Manager {
		c.GetUniqIds(&uniqIds)
	}
	ret := &netsvrProtocol.UniqIdListResp{}
	ret.CtxData = payload.CtxData
	ret.UniqIds = uniqIds
	route := &netsvrProtocol.Router{}
	route.Cmd = netsvrProtocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
