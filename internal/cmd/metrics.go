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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v2/netsvr"
	"netsvr/configs"
	"netsvr/internal/metrics"
	workerManager "netsvr/internal/worker/manager"
)

// Metrics 返回网关统计的服务状态
func Metrics(_ []byte, processor *workerManager.ConnProcessor) {
	ret := &netsvrProtocol.MetricsResp{}
	ret.ServerId = int32(configs.Config.ServerId)
	ret.Items = map[int32]*netsvrProtocol.MetricsRespItem{}
	for _, v := range metrics.Registry {
		if tmp := v.ToStatusResp(); tmp != nil {
			ret.Items[int32(v.Item)] = tmp
		}
	}
	processor.Send(ret, netsvrProtocol.Cmd_Metrics)
}
