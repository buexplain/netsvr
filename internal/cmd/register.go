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
	"google.golang.org/protobuf/proto"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/constant"
	"netsvr/pkg/protocol"
)

// Register 注册business进程
func Register(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.Register{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.Register failed")
		return
	}
	//检查服务编号是否在允许的范围内
	if constant.MinWorkerId > payload.Id || payload.Id > constant.MaxWorkerId {
		log.Logger.Error().Int32("workerId", payload.Id).Int("minWorkerId", constant.MinWorkerId).Int("maxWorkerId", constant.MaxWorkerId).Msg("WorkerId range overflow")
		processor.ForceClose()
		return
	}
	//检查当前的business连接是否已经注册过服务编号了，不允许重复注册
	if processor.GetWorkerId() > 0 {
		processor.ForceClose()
		log.Logger.Error().Msg("Register are not allowed")
		return
	}
	//设置business连接的服务编号
	processor.SetWorkerId(int(payload.Id))
	//将该服务编号登记到worker管理器中
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
