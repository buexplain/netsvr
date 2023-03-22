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
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/timer"
	workerManager "netsvr/internal/worker/manager"
	"time"
)

// ForceOfflineGuest 强制关闭某个空session值的连接
func ForceOfflineGuest(param []byte, _ *workerManager.ConnProcessor) {
	payload := &netsvrProtocol.ForceOfflineGuest{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.ForceOfflineGuest failed")
		return
	}
	if len(payload.UniqIds) == 0 {
		return
	}
	f := func(uniqId string) {
		conn := customerManager.Manager.Get(uniqId)
		if conn == nil {
			return
		}
		//跳过有session值的连接
		if session, ok := conn.Session().(*info.Info); ok && session.GetSession() != "" {
			return
		}
		//判断是否转发数据
		if len(payload.Data) == 0 {
			//无须转发任何数据，直接关闭连接
			_ = conn.Close()
		} else {
			//写入数据，并在一定倒计时后关闭连接
			_ = conn.WriteMessage(websocket.TextMessage, payload.Data)
			timer.Timer.AfterFunc(time.Second*3, func() {
				defer func() {
					_ = recover()
				}()
				_ = conn.Close()
			})
		}
	}
	if payload.Delay <= 0 {
		for _, uniqId := range payload.UniqIds {
			f(uniqId)
		}
	} else {
		timer.Timer.AfterFunc(time.Second*time.Duration(payload.Delay), func() {
			for _, uniqId := range payload.UniqIds {
				f(uniqId)
			}
		})
	}
}
