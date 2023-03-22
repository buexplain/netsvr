/**
* Copyright 2022 buexplain@qq.com
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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/protocol"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/limit"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// Limit 更新限流配置、获取网关中的限流配置的真实情况
func Limit(param []byte, processor *workerManager.ConnProcessor) {
	payload := netsvrProtocol.LimitReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.LimitReq failed")
		return
	}
	//更新限流配置
	if len(payload.Items) > 0 {
		for _, item := range payload.Items {
			limit.Manager.SetLimits(item.Num, item.WorkerIds)
		}
	}
	//返回网关中的限流配置的真实情况
	ret := &netsvrProtocol.LimitResp{}
	ret.CtxData = payload.CtxData
	ret.Items = limit.Manager.Count()
	route := &netsvrProtocol.Router{}
	route.Cmd = netsvrProtocol.Cmd(payload.RouterCmd)
	route.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(route)
	processor.Send(pt)
}
