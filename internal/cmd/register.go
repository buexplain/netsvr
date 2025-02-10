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
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"google.golang.org/protobuf/proto"
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
		ret.Message = ret.Code.String()
		processor.Send(ret, netsvrProtocol.Cmd_Register)
		return
	}
	//判断设置的事件是否有效
	invalidEvent := true
	for _, v := range netsvrProtocol.Event_value {
		if v == int32(netsvrProtocol.Event_Placeholder) {
			continue
		}
		if payload.Events&v == v {
			invalidEvent = false
			break
		}
	}
	if invalidEvent {
		ret.Code = netsvrProtocol.RegisterRespCode_InvalidEvent
		log.Logger.Error().Int32("events", payload.Events).Msg("Invalid RegisterReqEvent")
		ret.Message = ret.Code.String()
		processor.Send(ret, netsvrProtocol.Cmd_Register)
		return
	}
	//检查当前的business连接是否已经注册，不允许重复注册
	if processor.GetEvents() > 0 {
		ret.Code = netsvrProtocol.RegisterRespCode_DuplicateRegister
		log.Logger.Error().Msg("Duplicate register are not allowed")
		ret.Message = ret.Code.String()
		processor.Send(ret, netsvrProtocol.Cmd_Register)
		return
	}
	//设置business连接的events
	processor.SetEvents(payload.Events)
	//判断该business连接是否要开启更多的协程去处理它发来的请求命令
	if payload.ProcessCmdGoroutineNum > 1 {
		var i uint32 = 1
		for ; i < payload.ProcessCmdGoroutineNum; i++ {
			//添加到进程结束时的等待中，这样business发来的数据都会被处理完毕
			quit.Wg.Add(1)
			go processor.LoopCmd()
		}
	}
	//先将注册结果返回给business
	ret.Code = netsvrProtocol.RegisterRespCode_Success
	ret.Message = ret.Code.String()
	ret.ConnId = processor.GetConnId()
	processor.Send(ret, netsvrProtocol.Cmd_Register)
	//记录日志
	log.Logger.Info().
		Str("remoteAddr", processor.GetConnRemoteAddr()).
		Int32("events", payload.Events).
		Int("processCmdGoroutineNum", int(payload.ProcessCmdGoroutineNum)).
		Str("connId", processor.GetConnId()).
		Msg("Register a business")
	//最后，将该business连接登记到worker管理器中，一定要最后加入，确保注册结果的数据是第一个到达business进程，因为加入管理器后随时可能被转发数据
	workerManager.Manager.Set(processor)
}
