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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v3/netsvr"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// ConnInfo 获取uniqId的连接信息
func ConnInfo(param []byte, processor *workerManager.ConnProcessor) {
	payload := &netsvrProtocol.ConnInfoReq{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.ConnInfoReq failed")
		return
	}
	ret := &netsvrProtocol.ConnInfoResp{Items: map[string]*netsvrProtocol.ConnInfoRespItem{}}
	for _, uniqId := range payload.UniqIds {
		conn := customerManager.Manager.Get(uniqId)
		if conn == nil {
			continue
		}
		session, ok := conn.SessionWithLock().(*info.Info)
		if !ok {
			continue
		}
		item := &netsvrProtocol.ConnInfoRespItem{}
		session.GetConnInfoOnSafe(payload, item)
		ret.Items[uniqId] = item
	}
	processor.Send(ret, netsvrProtocol.Cmd_ConnInfo)
}
