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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"google.golang.org/protobuf/proto"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// CheckOnline 检查网关中是否包含某几个uniqId
func CheckOnline(param []byte, processor *workerManager.ConnProcessor) {
	payload := netsvrProtocol.CheckOnlineReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.CheckOnlineReq failed")
		return
	}
	ret := &netsvrProtocol.CheckOnlineResp{}
	uniqIds := make([]string, 0, len(payload.UniqIds))
	for _, uniqId := range payload.UniqIds {
		if customerManager.Manager.Has(uniqId) {
			uniqIds = append(uniqIds, uniqId)
		}
	}
	ret.UniqIds = uniqIds
	processor.Send(ret, netsvrProtocol.Cmd_CheckOnline)
}
