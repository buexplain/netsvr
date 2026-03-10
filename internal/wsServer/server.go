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

package wsServer

import (
	"context"
	"errors"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
	"io"
	"net/http"
	"time"
)

type Server struct {
	gnet.BuiltinEventEngine
	eng gnet.Engine
	//握手之前回调，可以返回http.Response，拒绝握手，在这回调中，你可以校验host、校验path等
	OnUpgradeCheck func(req *http.Request) *http.Response
	//握手成功后回调
	OnWebsocketOpen func(conn gnet.Conn, req *http.Request) (ws.StatusCode, error)
	//服务器响应ping后的回调
	OnWebsocketPing func(conn gnet.Conn)
	//收到数据帧的回调
	OnWebsocketMessage func(conn gnet.Conn, messageType ws.OpCode, messagePtr []byte)
	//服务器回显客户端的状态码和原因后的回调
	OnWebsocketClose func(conn gnet.Conn)
}

func (server *Server) OnlineNum() int {
	return server.eng.CountConnections()
}

func (server *Server) Shutdown(ctx context.Context) error {
	return server.eng.Stop(ctx)
}

func (server *Server) OnBoot(eng gnet.Engine) gnet.Action {
	server.eng = eng
	return gnet.None
}

func (server *Server) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	wsCodec := new(Codec)
	c.SetContext(wsCodec)
	wsCodec.currMessagePayload = NewMessagePayload(4)
	wsCodec.preMessageOpCode = 255
	wsCodec.currMessageOpCode = 255
	return nil, gnet.None
}

func (server *Server) OnClose(conn gnet.Conn, _ error) (action gnet.Action) {
	server.OnWebsocketClose(conn)
	return gnet.None
}

func (server *Server) OnTick() (delay time.Duration, action gnet.Action) {
	return time.Second, gnet.None
}

func (server *Server) OnTraffic(c gnet.Conn) (action gnet.Action) {
	wsCodec, _ := c.Context().(*Codec)
	if wsCodec.upgraded == false {
		var req *http.Request
		req, action = wsCodec.upgrade(server.OnUpgradeCheck, c)
		if wsCodec.upgraded {
			if statusCode, err := server.OnWebsocketOpen(c, req); err != nil {
				payload := ws.NewCloseFrameBody(statusCode, err.Error())
				wsCodec.SetClosed()
				_ = wsutil.WriteServerMessage(c, ws.OpClose, payload)
				return gnet.Close
			}
		}
		return action
	}
loop:
	completeMessagePayload, statusCode, err := wsCodec.decode(c)
	if err != nil {
		if !statusCode.Empty() {
			//数据帧解析错误，立即构造并发送close帧
			payload := ws.NewCloseFrameBody(statusCode, err.Error())
			wsCodec.SetClosed()
			_ = wsutil.WriteServerMessage(c, ws.OpClose, payload)
			return gnet.Close
		}
		//等待更多数据
		if errors.Is(err, io.ErrShortBuffer) {
			return gnet.None
		}
		//这行代码应该不会执行
		return gnet.Close
	}
	if wsCodec.currMessageOpCode.IsData() {
		server.OnWebsocketMessage(c, wsCodec.currMessageOpCode, completeMessagePayload)
		wsCodec.resetCurrMessage()
		if c.InboundBuffered() > 0 {
			//缓冲区还有数据，继续解析
			goto loop
		}
		return gnet.None
	} else if wsCodec.currMessageOpCode == ws.OpClose {
		// 是否服务端主动关闭，服务端主动关闭，则不能再回close包，否则客户端会报错：Close received after close
		if wsCodec.IsClosed() == false {
			//返回close，回显客户端的状态码和原因
			_ = wsutil.WriteServerMessage(c, ws.OpClose, completeMessagePayload)
		}
		return gnet.Close
	} else if wsCodec.currMessageOpCode == ws.OpPing {
		//返回pong，并将ping的payload一并返回
		err = wsutil.WriteServerMessage(c, ws.OpPong, completeMessagePayload)
		if err != nil {
			return gnet.Close
		}
		server.OnWebsocketPing(c)
		wsCodec.resetCurrMessage()
		if c.InboundBuffered() > 0 {
			goto loop
		}
		return gnet.None
	} else if wsCodec.currMessageOpCode == ws.OpPong {
		//不处理
		wsCodec.resetCurrMessage()
		if c.InboundBuffered() > 0 {
			//缓冲区还有数据，继续解析
			goto loop
		}
		return gnet.None
	} else if wsCodec.currMessageOpCode.IsReserved() {
		//不支持的数据帧
		payload := ws.NewCloseFrameBody(ws.StatusUnsupportedData, "unsupported opcode")
		wsCodec.SetClosed()
		_ = wsutil.WriteServerMessage(c, ws.OpClose, payload)
		return gnet.Close
	}
	//这块代码可能执行不到
	wsCodec.resetCurrMessage()
	if c.InboundBuffered() > 0 {
		//继续处理数据
		goto loop
	}
	return gnet.None
}
