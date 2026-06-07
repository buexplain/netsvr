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

package internal

import (
	"encoding/binary"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/gobwas/ws"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
)

func FormatSendToBusinessData(cmdBytes []byte, body []byte, event *zerolog.Event) *zerolog.Event {
	cmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(cmdBytes))
	if cmd == netsvrProtocol.Cmd_Transfer {
		tf := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(body, tf); err != nil {
			return event
		}
		event = event.Str("cmd", cmd.String()).Str("uniqId", tf.UniqId).
			Str("session", tf.Session).
			Str("customerId", tf.CustomerId).
			Strs("topics", tf.Topics)
		if configs.Config.Customer.SendMessageType == ws.OpText {
			return event.Str("data", string(tf.Data))
		}
		return event.Hex("dataHex", tf.Data)
	}
	if cmd == netsvrProtocol.Cmd_ConnOpen {
		co := &netsvrProtocol.ConnOpen{}
		if err := proto.Unmarshal(body, co); err != nil {
			return event
		}
		return event.Str("cmd", cmd.String()).Str("uniqId", co.UniqId).
			Str("rawQuery", co.RawQuery).
			Str("xForwardedFor", co.XForwardedFor).
			Str("xRealIp", co.XRealIp).
			Str("remoteAddr", co.RemoteAddr)
	}
	if cmd == netsvrProtocol.Cmd_ConnClose {
		cc := &netsvrProtocol.ConnClose{}
		if err := proto.Unmarshal(body, cc); err != nil {
			return event
		}
		return event.Str("cmd", cmd.String()).Str("uniqId", cc.UniqId).
			Str("customerId", cc.CustomerId).
			Str("session", cc.Session).
			Strs("topics", cc.Topics)
	}
	//非客户端的命令，只打印cmd
	return event.Str("cmd", cmd.String())
}
