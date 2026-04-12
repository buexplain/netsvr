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

package binder

import (
	"github.com/panjf2000/gnet/v2"
	"io"
	"net"
	"time"
)

// MockConn 模拟 gnet.Conn 用于测试
type MockConn struct {
	context   any
	eventLoop gnet.EventLoop
}

func (m *MockConn) Context() any {
	return m.context
}

func (m *MockConn) SetContext(context any) {
	m.context = context
}

func (m *MockConn) EventLoop() gnet.EventLoop {
	return m.eventLoop
}

// 实现 gnet.Conn 的其他必需方法（空实现）
func (m *MockConn) AsyncWrite([]byte, gnet.AsyncCallback) error                { return nil }
func (m *MockConn) AsyncWritev([][]byte, gnet.AsyncCallback) error             { return nil }
func (m *MockConn) Wake(gnet.AsyncCallback) error                              { return nil }
func (m *MockConn) Close() error                                               { return nil }
func (m *MockConn) CloseWithCallback(gnet.AsyncCallback) error                 { return nil }
func (m *MockConn) Flush() error                                               { return nil }
func (m *MockConn) Next(int) ([]byte, error)                                   { return nil, nil }
func (m *MockConn) Read([]byte) (int, error)                                   { return 0, nil }
func (m *MockConn) WriteTo(io.Writer) (int64, error)                           { return 0, nil }
func (m *MockConn) ReadFrom(io.Reader) (int64, error)                          { return 0, nil }
func (m *MockConn) SendTo([]byte, net.Addr) (int, error)                       { return 0, nil }
func (m *MockConn) ResetBuffer()                                               {}
func (m *MockConn) ReadN(_ int) ([]byte, error)                                { return nil, nil }
func (m *MockConn) BufferLength() int                                          { return 0 }
func (m *MockConn) InboundBuffered() int                                       { return 0 }
func (m *MockConn) OutboundBuffered() int                                      { return 0 }
func (m *MockConn) Write([]byte) (int, error)                                  { return 0, nil }
func (m *MockConn) Writev([][]byte) (int, error)                               { return 0, nil }
func (m *MockConn) Peek(int) ([]byte, error)                                   { return nil, nil }
func (m *MockConn) Discard(int) (int, error)                                   { return 0, nil }
func (m *MockConn) ShiftN(_ int) (int, error)                                  { return 0, nil }
func (m *MockConn) LocalAddr() net.Addr                                        { return nil }
func (m *MockConn) RemoteAddr() net.Addr                                       { return nil }
func (m *MockConn) Fd() int                                                    { return 0 }
func (m *MockConn) Dup() (int, error)                                          { return 0, nil }
func (m *MockConn) SetReadBuffer(int) error                                    { return nil }
func (m *MockConn) SetWriteBuffer(int) error                                   { return nil }
func (m *MockConn) SetLinger(int) error                                        { return nil }
func (m *MockConn) SetNoDelay(bool) error                                      { return nil }
func (m *MockConn) SetKeepAlivePeriod(time.Duration) error                     { return nil }
func (m *MockConn) SetKeepAlive(bool, time.Duration, time.Duration, int) error { return nil }
func (m *MockConn) SetDeadline(time.Time) error                                { return nil }
func (m *MockConn) SetReadDeadline(time.Time) error                            { return nil }
func (m *MockConn) SetWriteDeadline(time.Time) error                           { return nil }

// newTestCollect 创建一个新的 collect 实例用于测试
func newTestCollect() *collect {
	c := &collect{}
	for i := range c.shards {
		c.shards[i].data = make(map[string]map[int]gnet.Conn, shardCount)
	}
	return c
}
