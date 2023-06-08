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
	"math"
	"netsvr/configs"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/quit"
)

// Register 注册business进程
func Register(param []byte, processor *workerManager.ConnProcessor) {
	payload := netsvrProtocol.RegisterReq{}
	ret := &netsvrProtocol.RegisterResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		ret.Code = netsvrProtocol.RegisterRespCode_UnmarshalError
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.RegisterReq failed")
		goto RESPONSE
	}
	//检查workerId是否在允许的范围内
	if netsvrProtocol.WorkerIdMin > payload.WorkerId || payload.WorkerId > netsvrProtocol.WorkerIdMax {
		ret.Code = netsvrProtocol.RegisterRespCode_WorkerIdOverflow
		log.Logger.Error().Int32("workerId", payload.WorkerId).Int("workerIdMin", netsvrProtocol.WorkerIdMin).Int("workerIdMax", netsvrProtocol.WorkerIdMax).Msg("WorkerId range overflow")
		goto RESPONSE
	}
	//检查business报出的serverId是否与本网关配置的serverId一致，如果不一致，则断开连接
	if payload.ServerId > math.MaxUint8 || uint8(payload.ServerId) != configs.Config.ServerId {
		ret.Code = netsvrProtocol.RegisterRespCode_ServerIdInconsistent
		log.Logger.Error().Uint8("correctServerId", configs.Config.ServerId).Uint32("errorServerId", payload.ServerId).Msg("ServerId is error")
		goto RESPONSE
	}
	//检查当前的business连接是否已经注册过workerId了，不允许重复注册
	if processor.GetWorkerId() > 0 {
		ret.Code = netsvrProtocol.RegisterRespCode_DuplicateRegister
		log.Logger.Error().Msg("Duplicate register are not allowed")
		goto RESPONSE
	}
	//设置business连接的workerId
	processor.SetWorkerId(payload.WorkerId)
	//将该workerId登记到worker管理器中
	workerManager.Manager.Set(processor.GetWorkerId(), processor)
	//判断该business连接是否要开启更多的协程去处理它发来的请求命令
	if payload.ProcessCmdGoroutineNum > 1 {
		var i uint32 = 1
		for ; i < payload.ProcessCmdGoroutineNum; i++ {
			//添加到进程结束时的等待中，这样business发来的数据都会被处理完毕
			quit.Wg.Add(1)
			go processor.LoopCmd()
		}
	}
	log.Logger.Info().Int32("workerId", payload.WorkerId).Msg("Register a business")
	ret.Code = netsvrProtocol.RegisterRespCode_Success
RESPONSE:
	ret.Message = ret.Code.String()
	processor.Send(ret, netsvrProtocol.Cmd_Register)
}
