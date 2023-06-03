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

// Package wsTimer 定时器模块
// 提供定时间隔或者是延时处理服务
package wsTimer

import (
	gTimer "github.com/antlabs/timer"
	"netsvr/pkg/quit"
)

var WsTimer gTimer.Timer

func init() {
	WsTimer = gTimer.NewTimer()
	go WsTimer.Run()
	//收到进程结束信号，则立马停止定时器
	go func() {
		defer func() {
			_ = recover()
		}()
		<-quit.ClosedCh
		WsTimer.Stop()
	}()
}
