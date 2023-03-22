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
	"google.golang.org/protobuf/proto"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// Register 注册business进程
func Register(param []byte, processor *workerManager.ConnProcessor) {
	payload := netsvrProtocol.Register{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.Register failed")
		return
	}
	//检查workerId是否在允许的范围内
	if netsvrProtocol.WorkerIdMin > payload.Id || payload.Id > netsvrProtocol.WorkerIdMax {
		log.Logger.Error().Int32("workerId", payload.Id).Int("workerIdMin", netsvrProtocol.WorkerIdMin).Int("workerIdMax", netsvrProtocol.WorkerIdMax).Msg("WorkerId range overflow")
		processor.ForceClose()
		return
	}
	//检查当前的business连接是否已经注册过workerId了，不允许重复注册
	if processor.GetWorkerId() > 0 {
		processor.ForceClose()
		log.Logger.Error().Msg("Register are not allowed")
		return
	}
	//设置business连接的workerId
	processor.SetWorkerId(payload.Id)
	//将该workerId登记到worker管理器中
	workerManager.Manager.Set(processor.GetWorkerId(), processor)
	//判断该business连接是否要开启更多的协程去处理它发来的请求命令
	if payload.ProcessCmdGoroutineNum > 1 {
		var i uint32 = 1
		for ; i < payload.ProcessCmdGoroutineNum; i++ {
			go processor.LoopCmd()
		}
	}
	log.Logger.Debug().Int32("workerId", payload.Id).Msg("Register a business")
}
