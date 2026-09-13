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

// 本文件是测试客户端：连接网关、收发消息，以及建立在它之上的接收断言。
package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"netsvr/test/pkg/protocol"
)

// readTimeout 单条消息的读取超时。网关到业务进程再到客户这条链路都在本机，5 秒足够
const readTimeout = 5 * time.Second

// wsClient 一个连上网关的客户。
// 读取由独立的 goroutine 负责并投递到 channel，测试侧只做「带超时的接收」。
// 这样断言「收不到数据」时不必去改连接的读超时——gorilla/websocket 的读一旦超时，
// 这条连接后续读取会永久失败，不能再复用。
type wsClient struct {
	t         *testing.T
	conn      *websocket.Conn
	uniqId    string
	incoming  chan incomingMessage
	done      chan struct{}
	closeOnce sync.Once
}

// incomingMessage 读取 goroutine 投递给测试侧的一条消息或一个错误
type incomingMessage struct {
	cmd protocol.Cmd
	env *envelope
	err error
}

// ============================== 建连 ==============================

// newWsClient 连上网关并等待业务进程推回连接打开回包，从中取出 uniqId
func newWsClient(t *testing.T) *wsClient {
	t.Helper()
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("连接网关 %s 失败：%v", wsURL, err)
	}
	c := &wsClient{
		t:        t,
		conn:     conn,
		incoming: make(chan incomingMessage, 64),
		done:     make(chan struct{}),
	}
	t.Cleanup(c.close)
	go c.readLoop()
	cmd, env := c.read()
	if cmd != protocol.RouterRespConnOpen {
		t.Fatalf("连接建立后第一条消息应为连接打开回包，实际 cmd=%d", cmd)
	}
	requireOK(t, env)
	var payload connOpenPayload
	env.into(t, &payload)
	if payload.UniqId == "" {
		t.Fatalf("连接打开回包里没有 uniqId：%s", env.Data)
	}
	c.uniqId = payload.UniqId
	return c
}

// close 关闭连接与读取 goroutine，可重复调用
func (c *wsClient) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

// ============================== 读取 ==============================

// readLoop 这条连接唯一的读取方：解析后投递，出错则投递错误并退出。
// 注意不能在这里调用 t.Fatalf（非测试 goroutine）
func (c *wsClient) readLoop() {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.deliver(incomingMessage{err: err})
			return
		}
		var msg respMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.deliver(incomingMessage{err: fmt.Errorf("解析网关数据失败：%v（原始：%s）", err, data)})
			return
		}
		env := &envelope{}
		if len(msg.Data) > 0 {
			if err := json.Unmarshal(msg.Data, env); err != nil {
				c.deliver(incomingMessage{err: fmt.Errorf("解析网关数据的 data 失败：%v（原始：%s）", err, msg.Data)})
				return
			}
		}
		c.deliver(incomingMessage{cmd: msg.Cmd, env: env})
	}
}

func (c *wsClient) deliver(m incomingMessage) {
	select {
	case c.incoming <- m:
	case <-c.done:
	}
}

// receive 在超时时间内等待下一条消息，第二个返回值表示是否等到
func (c *wsClient) receive(timeout time.Duration) (incomingMessage, bool) {
	select {
	case m := <-c.incoming:
		return m, true
	case <-time.After(timeout):
		return incomingMessage{}, false
	}
}

// read 读一条消息，超时或出错都直接判定用例失败
func (c *wsClient) read() (protocol.Cmd, *envelope) {
	c.t.Helper()
	m, ok := c.receive(readTimeout)
	if !ok {
		c.t.Fatalf("等待网关数据超时")
	}
	if m.err != nil {
		c.t.Fatalf("读取网关数据失败：%v", m.err)
	}
	return m.cmd, m.env
}

// waitFor 读到指定 cmd 的回包为止；读到的其它 cmd 视为异常
func (c *wsClient) waitFor(cmd protocol.Cmd) *envelope {
	c.t.Helper()
	gotCmd, env := c.read()
	if gotCmd != cmd {
		c.t.Fatalf("回包 cmd 不符合预期：期望 %s(%d)，实际 %s(%d)",
			protocol.CmdName[cmd], cmd, protocol.CmdName[gotCmd], gotCmd)
	}
	return env
}

// waitClose 一直读到连接被关闭，返回关闭码；期间收到的连接关闭投递内容一并返回
func (c *wsClient) waitClose() (int, []string) {
	c.t.Helper()
	var received []string
	for {
		m, ok := c.receive(readTimeout)
		if !ok {
			c.t.Fatalf("等待关闭帧超时")
		}
		if m.err != nil {
			var closeErr *websocket.CloseError
			if errors.As(m.err, &closeErr) {
				return closeErr.Code, received
			}
			c.t.Fatalf("等待关闭帧失败：%v", m.err)
		}
		if m.cmd == protocol.RouterRespConnClose {
			received = append(received, m.env.Message)
		}
	}
}

// ============================== 发送 ==============================

// send 发送一条命令，不等待回包
func (c *wsClient) send(cmd protocol.Cmd, param any) {
	c.t.Helper()
	payload := ""
	if param != nil {
		b, err := json.Marshal(param)
		if err != nil {
			c.t.Fatalf("序列化命令参数失败：%v", err)
		}
		payload = string(b)
	}
	b, err := json.Marshal(clientCmd{Cmd: cmd, Data: payload})
	if err != nil {
		c.t.Fatalf("序列化命令失败：%v", err)
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
		c.t.Fatalf("发送命令 %s 失败：%v", protocol.CmdName[cmd], err)
	}
}

// call 发送命令并等待同 cmd 的回包（不校验 code）
func (c *wsClient) call(cmd protocol.Cmd, param any) *envelope {
	c.t.Helper()
	c.send(cmd, param)
	return c.waitFor(cmd)
}

// callOK 发送命令并等待回包，要求 code == 0
func (c *wsClient) callOK(cmd protocol.Cmd, param any) *envelope {
	c.t.Helper()
	env := c.call(cmd, param)
	requireOK(c.t, env)
	return env
}

// ============================== 接收断言 ==============================

// assertNoMessage 断言连接在给定时间内收不到任何数据。
// 用于验证「目标不存在 / 目标为空 / 数据为空」时不会产生多余投递
func (c *wsClient) assertNoMessage(timeout time.Duration) {
	c.t.Helper()
	m, ok := c.receive(timeout)
	if !ok {
		return
	}
	if m.err != nil {
		c.t.Fatalf("连接不应断开，实际：%v", m.err)
	}
	c.t.Fatalf("连接不应收到数据，实际 cmd=%d", m.cmd)
}

// expectCast 读取一条投递消息并断言其 cmd 与内容
func (c *wsClient) expectCast(cmd protocol.Cmd, want fromUserMessage) {
	c.t.Helper()
	gotCmd, env := c.read()
	if gotCmd != cmd {
		c.t.Fatalf("投递消息的 cmd 不符合预期：期望 %s(%d)，实际 %s(%d)",
			protocol.CmdName[cmd], cmd, protocol.CmdName[gotCmd], gotCmd)
	}
	requireOK(c.t, env)
	var got fromUserMessage
	env.into(c.t, &got)
	if got.Message != want.Message {
		c.t.Fatalf("投递内容不符合预期：期望 %q，实际 %q", want.Message, got.Message)
	}
	if want.FromUser != "" && got.FromUser != want.FromUser {
		c.t.Fatalf("投递来源不符合预期：期望 %q，实际 %q", want.FromUser, got.FromUser)
	}
}
