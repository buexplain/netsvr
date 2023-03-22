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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// Unregister business取消已注册的workerId
func Unregister(_ []byte, processor *workerManager.ConnProcessor) {
	workerId := processor.GetWorkerId()
	if netsvrProtocol.WorkerIdMin <= workerId && workerId <= netsvrProtocol.WorkerIdMax {
		workerManager.Manager.Del(workerId, processor)
		log.Logger.Info().Int32("workerId", workerId).Msg("Unregister a business")
	}
}
