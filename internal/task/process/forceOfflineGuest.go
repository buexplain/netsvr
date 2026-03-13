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

package process

import (
	"errors"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/gobwas/ws"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	"netsvr/internal/customer"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	"netsvr/internal/timer"
	"time"
)

// forceOfflineGuest 强制关闭某个空session值的连接
func forceOfflineGuest(param []byte) {
	payload := &netsvrProtocol.ForceOfflineGuest{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.forceOfflineGuest failed")
		return
	}
	if len(payload.UniqIds) == 0 {
		return
	}
	closeFrame := customer.BuildCloseFrame(ws.StatusPolicyViolation, errors.New("forceOfflineGuest"))
	fn := func(uniqId string, msg *customer.Message) {
		conn := customerManager.Manager.Get(uniqId)
		session, ok := customer.GetSession(conn)
		//跳过无效的 session，或者已经登录的 session
		if !ok || session.IsLogin() {
			return
		}
		//判断是否转发数据
		if msg != nil {
			msg.WriteTo(conn)
		}
		customer.WriteCloseFrame(conn, closeFrame)
	}
	if payload.Delay <= 0 {
		var msg *customer.Message
		if len(payload.Data) > 0 {
			msg = customer.NewMessage(configs.Config.Customer.SendMessageType, payload.Data)
		}
		for _, uniqId := range payload.UniqIds {
			fn(uniqId, msg)
		}
	} else {
		timer.Timer.AfterFunc(time.Second*time.Duration(payload.Delay), func() {
			defer func() {
				_ = recover()
			}()
			var msg *customer.Message
			if len(payload.Data) > 0 {
				msg = customer.NewMessage(configs.Config.Customer.SendMessageType, payload.Data)
			}
			for _, uniqId := range payload.UniqIds {
				fn(uniqId, msg)
			}
		})
	}
}
