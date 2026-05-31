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

package customer

import (
	"github.com/gobwas/ws"
	"github.com/panjf2000/gnet/v2"
	"io"
	"net"
	"netsvr/internal/wsServer"
	"sync"
	"testing"
	"time"
)

// MockGnetConnForFrame 是 gnet.Conn 的 mock 实现，用于 Frame 测试
type MockGnetConnForFrame struct {
	id          uint64
	localAddr   net.Addr
	remoteAddr  net.Addr
	writeData   [][]byte
	closeCalled bool
	context     interface{}
	eventLoop   gnet.EventLoop
	closed      bool
	compress    bool
	mu          sync.Mutex
}

func NewMockGnetConnForFrame(id uint64) *MockGnetConnForFrame {
	return &MockGnetConnForFrame{
		id:         id,
		localAddr:  &mockAddrFrame{network: "tcp", addr: "127.0.0.1:8080"},
		remoteAddr: &mockAddrFrame{network: "tcp", addr: "127.0.0.1:9090"},
		writeData:  make([][]byte, 0),
		closed:     false,
		compress:   false,
	}
}

func (m *MockGnetConnForFrame) Read(_ []byte) (n int, err error)         { return 0, nil }
func (m *MockGnetConnForFrame) WriteTo(_ io.Writer) (n int64, err error) { return 0, nil }
func (m *MockGnetConnForFrame) Next(_ int) (buf []byte, err error)       { return nil, nil }
func (m *MockGnetConnForFrame) Peek(_ int) (buf []byte, err error)       { return nil, nil }
func (m *MockGnetConnForFrame) Discard(_ int) (discarded int, err error) { return 0, nil }
func (m *MockGnetConnForFrame) InboundBuffered() int                     { return 0 }
func (m *MockGnetConnForFrame) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data := make([]byte, len(b))
	copy(data, b)
	m.writeData = append(m.writeData, data)
	return len(b), nil
}
func (m *MockGnetConnForFrame) ReadFrom(_ io.Reader) (n int64, err error)      { return 0, nil }
func (m *MockGnetConnForFrame) SendTo(_ []byte, _ net.Addr) (n int, err error) { return 0, nil }
func (m *MockGnetConnForFrame) Writev(_ [][]byte) (n int, err error)           { return 0, nil }
func (m *MockGnetConnForFrame) Flush() error                                   { return nil }
func (m *MockGnetConnForFrame) OutboundBuffered() int                          { return 0 }
func (m *MockGnetConnForFrame) AsyncWrite(buf []byte, callback gnet.AsyncCallback) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return io.ErrClosedPipe
	}
	data := make([]byte, len(buf))
	copy(data, buf)
	m.writeData = append(m.writeData, data)
	if callback != nil {
		_ = callback(m, nil)
	}
	return nil
}
func (m *MockGnetConnForFrame) AsyncWritev(_ [][]byte, callback gnet.AsyncCallback) error {
	if callback != nil {
		_ = callback(m, nil)
	}
	return nil
}
func (m *MockGnetConnForFrame) Fd() int                                              { return 0 }
func (m *MockGnetConnForFrame) Dup() (int, error)                                    { return 0, nil }
func (m *MockGnetConnForFrame) SetReadBuffer(_ int) error                            { return nil }
func (m *MockGnetConnForFrame) SetWriteBuffer(_ int) error                           { return nil }
func (m *MockGnetConnForFrame) SetLinger(_ int) error                                { return nil }
func (m *MockGnetConnForFrame) SetKeepAlivePeriod(_ time.Duration) error             { return nil }
func (m *MockGnetConnForFrame) SetKeepAlive(_ bool, _, _ time.Duration, _ int) error { return nil }
func (m *MockGnetConnForFrame) SetNoDelay(_ bool) error                              { return nil }
func (m *MockGnetConnForFrame) Context() interface{}                                 { return m.context }
func (m *MockGnetConnForFrame) EventLoop() gnet.EventLoop                            { return m.eventLoop }
func (m *MockGnetConnForFrame) SetContext(ctx interface{})                           { m.context = ctx }
func (m *MockGnetConnForFrame) LocalAddr() net.Addr                                  { return m.localAddr }
func (m *MockGnetConnForFrame) RemoteAddr() net.Addr                                 { return m.remoteAddr }
func (m *MockGnetConnForFrame) Wake(_ gnet.AsyncCallback) error                      { return nil }
func (m *MockGnetConnForFrame) CloseWithCallback(callback gnet.AsyncCallback) error {
	m.mu.Lock()
	m.closed = true
	m.closeCalled = true
	m.mu.Unlock()
	if callback != nil {
		_ = callback(m, nil)
	}
	return nil
}
func (m *MockGnetConnForFrame) Close() error {
	m.mu.Lock()
	m.closed = true
	m.closeCalled = true
	m.mu.Unlock()
	return nil
}
func (m *MockGnetConnForFrame) SetDeadline(_ time.Time) error      { return nil }
func (m *MockGnetConnForFrame) SetReadDeadline(_ time.Time) error  { return nil }
func (m *MockGnetConnForFrame) SetWriteDeadline(_ time.Time) error { return nil }

// GetWriteData 获取写入的数据（线程安全）
func (m *MockGnetConnForFrame) GetWriteData() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.writeData))
	for i, data := range m.writeData {
		result[i] = make([]byte, len(data))
		copy(result[i], data)
	}
	return result
}

// IsClosed 检查连接是否关闭（线程安全）
func (m *MockGnetConnForFrame) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// SetCompress 设置压缩标志
func (m *MockGnetConnForFrame) SetCompress(compress bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.compress = compress
}

// IsCompress 获取压缩标志
func (m *MockGnetConnForFrame) IsCompress() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.compress
}

type mockAddrFrame struct {
	network string
	addr    string
}

func (m *mockAddrFrame) Network() string { return m.network }
func (m *mockAddrFrame) String() string  { return m.addr }

// createTestWsConn 创建测试用的 wsServer.Conn
func createTestWsConn(id uint64) *wsServer.Conn {
	mockConn := NewMockGnetConnForFrame(id)
	return wsServer.NewConn(mockConn)
}

// createTestWsConnWithCompression 创建启用压缩的测试连接
func createTestWsConnWithCompression(id uint64, compress bool) *wsServer.Conn {
	mockConn := NewMockGnetConnForFrame(id)
	conn := wsServer.NewConn(mockConn)
	wsServer.SetCompressionForTest(conn, compress)
	return conn
}

// TestFrame_BasicGetAndPut 测试基本的 Get 和 Put
func TestFrame_BasicGetAndPut(t *testing.T) {
	messageType := ws.OpText
	data := []byte("Hello, WebSocket!")

	frame := FrameObjPool.Get(messageType, data)
	if frame == nil {
		t.Fatal("期望获取到 Frame, 实际为 nil")
	}

	if frame.messageType != messageType {
		t.Errorf("期望 messageType=%v, 实际=%v", messageType, frame.messageType)
	}

	if string(frame.data) != string(data) {
		t.Errorf("期望 data=%s, 实际=%s", string(data), string(frame.data))
	}

	FrameObjPool.Put(frame)

	if frame.data != nil {
		t.Error("Put 后 data 应该为 nil")
	}
	if frame.compressed != nil {
		t.Error("Put 后 compressed 应该为 nil")
	}
	if frame.uncompressed != nil {
		t.Error("Put 后 uncompressed 应该为 nil")
	}
}

// TestFrame_HeaderSize_Boundary125 测试边界值 125 字节（2 字节头）
func TestFrame_HeaderSize_Boundary125(t *testing.T) {
	conn := createTestWsConn(1)
	data := make([]byte, 125)
	for i := range data {
		data[i] = byte(i % 256)
	}

	frame := FrameObjPool.Get(ws.OpText, data)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("125 字节数据应该成功写入")
	}

	if frame.uncompressed == nil {
		t.Fatal("期望 uncompressed 被设置")
	}

	expectedSize := 2 + 125
	if len(frame.uncompressed) != expectedSize {
		t.Errorf("期望帧大小=%d, 实际=%d", expectedSize, len(frame.uncompressed))
	}
}

// TestFrame_HeaderSize_Boundary126 测试边界值 126 字节（4 字节头）
func TestFrame_HeaderSize_Boundary126(t *testing.T) {
	conn := createTestWsConn(1)
	data := make([]byte, 126)
	for i := range data {
		data[i] = byte(i % 256)
	}

	frame := FrameObjPool.Get(ws.OpText, data)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("126 字节数据应该成功写入")
	}

	if frame.uncompressed == nil {
		t.Fatal("期望 uncompressed 被设置")
	}

	expectedSize := 2 + 2 + 126
	if len(frame.uncompressed) != expectedSize {
		t.Errorf("期望帧大小=%d, 实际=%d", expectedSize, len(frame.uncompressed))
	}
}

// TestFrame_HeaderSize_Boundary65535 测试边界值 65535 字节（4 字节头）
func TestFrame_HeaderSize_Boundary65535(t *testing.T) {
	conn := createTestWsConn(1)
	data := make([]byte, 65535)
	for i := range data {
		data[i] = byte(i % 256)
	}

	frame := FrameObjPool.Get(ws.OpBinary, data)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("65535 字节数据应该成功写入")
	}

	if frame.uncompressed == nil {
		t.Fatal("期望 uncompressed 被设置")
	}

	expectedSize := 2 + 2 + 65535
	if len(frame.uncompressed) != expectedSize {
		t.Errorf("期望帧大小=%d, 实际=%d", expectedSize, len(frame.uncompressed))
	}
}

// TestFrame_HeaderSize_Boundary65536 测试边界值 65536 字节（10 字节头）
func TestFrame_HeaderSize_Boundary65536(t *testing.T) {
	conn := createTestWsConn(1)
	data := make([]byte, 65536)
	for i := range data {
		data[i] = byte(i % 256)
	}

	frame := FrameObjPool.Get(ws.OpBinary, data)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("65536 字节数据应该成功写入")
	}

	if frame.uncompressed == nil {
		t.Fatal("期望 uncompressed 被设置")
	}

	expectedSize := 2 + 8 + 65536
	if len(frame.uncompressed) != expectedSize {
		t.Errorf("期望帧大小=%d, 实际=%d", expectedSize, len(frame.uncompressed))
	}
}

// TestFrame_WriteTo_ClosedConnection 测试向已关闭的连接写入
func TestFrame_WriteTo_ClosedConnection(t *testing.T) {
	conn := createTestWsConn(1)

	// 先关闭连接
	mockGnetConn := conn.RemoteAddrOnSafe().(*mockAddrFrame)
	_ = mockGnetConn

	// 通过 wsServer.Conn 的方法关闭
	conn.CloseOnSafe()

	data := []byte("test message")
	frame := FrameObjPool.Get(ws.OpText, data)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != false {
		t.Error("向已关闭的连接写入应该返回 false")
	}
}

// TestFrame_WriteTo_Uncompressed 测试未压缩消息的写入
func TestFrame_WriteTo_Uncompressed(t *testing.T) {
	conn := createTestWsConn(1)

	data := []byte("small message")
	frame := FrameObjPool.Get(ws.OpText, data)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("期望写入成功，实际失败")
	}

	if frame.uncompressed == nil {
		t.Error("期望 uncompressed 被缓存，实际为 nil")
	}

	if frame.compressed != nil {
		t.Error("期望 compressed 为 nil，实际有值")
	}
}

// TestFrame_WriteTo_MultipleConnections 测试同一 Frame 向多个连接写入
func TestFrame_WriteTo_MultipleConnections(t *testing.T) {
	conn1 := createTestWsConn(1)
	conn2 := createTestWsConn(2)
	conn3 := createTestWsConn(3)

	data := []byte("broadcast message")
	frame := FrameObjPool.Get(ws.OpBinary, data)
	defer FrameObjPool.Put(frame)

	result1 := frame.WriteTo(conn1)
	if result1 != true {
		t.Error("向第一个连接写入应该成功")
	}

	firstUncompressed := frame.uncompressed
	if firstUncompressed == nil {
		t.Fatal("第一次写入后 uncompressed 应该有值")
	}

	result2 := frame.WriteTo(conn2)
	if result2 != true {
		t.Error("向第二个连接写入应该成功")
	}

	if &frame.uncompressed[0] != &firstUncompressed[0] {
		t.Error("第二次写入应该复用 uncompressed 缓存")
	}

	result3 := frame.WriteTo(conn3)
	if result3 != true {
		t.Error("向第三个连接写入应该成功")
	}
}

// TestFrame_PoolReuse 测试对象池的复用
func TestFrame_PoolReuse(t *testing.T) {
	frame1 := FrameObjPool.Get(ws.OpText, []byte("message1"))
	if frame1 == nil {
		t.Fatal("期望获取到 Frame")
	}

	FrameObjPool.Put(frame1)

	frame2 := FrameObjPool.Get(ws.OpText, []byte("message2"))
	if frame2 == nil {
		t.Fatal("期望再次获取到 Frame")
	}

	if string(frame2.data) != "message2" {
		t.Errorf("期望 data=message2, 实际=%s", string(frame2.data))
	}

	FrameObjPool.Put(frame2)
}

// TestFrame_DifferentMessageTypes 测试不同的消息类型
func TestFrame_DifferentMessageTypes(t *testing.T) {
	testCases := []struct {
		name        string
		messageType ws.OpCode
		data        []byte
	}{
		{"Text Message", ws.OpText, []byte("text content")},
		{"Binary Message", ws.OpBinary, []byte{0x00, 0x01, 0x02, 0x03}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := createTestWsConn(1)
			frame := FrameObjPool.Get(tc.messageType, tc.data)
			defer FrameObjPool.Put(frame)

			result := frame.WriteTo(conn)
			if result != true {
				t.Errorf("%s: 期望写入成功，实际失败", tc.name)
			}

			if frame.messageType != tc.messageType {
				t.Errorf("%s: 期望 messageType=%v, 实际=%v", tc.name, tc.messageType, frame.messageType)
			}
		})
	}
}

// TestFrame_EmptyData 测试空数据
func TestFrame_EmptyData(t *testing.T) {
	conn := createTestWsConn(1)
	var data []byte
	frame := FrameObjPool.Get(ws.OpText, data)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("空数据也应该能成功写入")
	}
}

// TestFrame_LargeData 测试大数据
func TestFrame_LargeData(t *testing.T) {
	conn := createTestWsConn(1)
	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i % 256)
	}

	frame := FrameObjPool.Get(ws.OpBinary, data)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("大数据也应该能成功写入")
	}

	if frame.uncompressed == nil {
		t.Error("期望 uncompressed 被设置")
	}
}

// TestFrame_VeryLargeData 测试超大数据（超过 65535 字节）
func TestFrame_VeryLargeData(t *testing.T) {
	conn := createTestWsConn(1)
	data := make([]byte, 70000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	frame := FrameObjPool.Get(ws.OpBinary, data)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("超大数据也应该能成功写入")
	}

	if frame.uncompressed == nil {
		t.Error("期望 uncompressed 被设置")
	}
}

// TestFrame_SetMethod 测试 set 方法
func TestFrame_SetMethod(t *testing.T) {
	frame := &Frame{}

	messageType := ws.OpBinary
	data := []byte("test data")

	frame.set(messageType, data)

	if frame.messageType != messageType {
		t.Errorf("期望 messageType=%v, 实际=%v", messageType, frame.messageType)
	}

	if string(frame.data) != string(data) {
		t.Errorf("期望 data=%s, 实际=%s", string(data), string(frame.data))
	}
}

// TestFrame_ResetMethod 测试 reset 方法
func TestFrame_ResetMethod(t *testing.T) {
	frame := &Frame{
		messageType:  ws.OpText,
		data:         []byte("test"),
		compressed:   []byte("compressed"),
		uncompressed: []byte("uncompressed"),
	}

	frame.reset()

	if frame.data != nil {
		t.Error("reset 后 data 应该为 nil")
	}
	if frame.compressed != nil {
		t.Error("reset 后 compressed 应该为 nil")
	}
	if frame.uncompressed != nil {
		t.Error("reset 后 uncompressed 应该为 nil")
	}
}

// TestFrame_CompressionThreshold 测试压缩阈值逻辑
func TestFrame_CompressionThreshold(t *testing.T) {
	conn := createTestWsConn(1)
	smallData := make([]byte, 100)
	frame := FrameObjPool.Get(ws.OpText, smallData)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("小数据应该成功写入")
	}

	if frame.compressed != nil {
		t.Error("小数据不应该设置 compressed 缓存")
	}
}

// TestFrame_WriteTo_Compressed 测试压缩消息的写入
func TestFrame_WriteTo_Compressed(t *testing.T) {
	conn := createTestWsConnWithCompression(1, true)

	largeData := make([]byte, 300)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	frame := FrameObjPool.Get(ws.OpBinary, largeData)
	defer FrameObjPool.Put(frame)

	result := frame.WriteTo(conn)
	if result != true {
		t.Error("压缩消息应该成功写入")
	}

	if frame.compressed == nil {
		t.Error("期望 compressed 被缓存，实际为 nil")
	}

	if frame.uncompressed != nil {
		t.Error("期望 uncompressed 为 nil，实际有值")
	}
}

// TestFrame_CompressedCacheReuse 测试压缩缓存的复用
func TestFrame_CompressedCacheReuse(t *testing.T) {
	conn1 := createTestWsConnWithCompression(1, true)
	conn2 := createTestWsConnWithCompression(2, true)
	conn3 := createTestWsConnWithCompression(3, true)

	data := make([]byte, 500)
	for i := range data {
		data[i] = byte(i % 10)
	}

	frame := FrameObjPool.Get(ws.OpText, data)
	defer FrameObjPool.Put(frame)

	result1 := frame.WriteTo(conn1)
	if result1 != true {
		t.Error("向第一个连接写入应该成功")
	}

	firstCompressed := frame.compressed
	if firstCompressed == nil {
		t.Fatal("第一次写入后 compressed 应该有值")
	}

	result2 := frame.WriteTo(conn2)
	if result2 != true {
		t.Error("向第二个连接写入应该成功")
	}

	if &frame.compressed[0] != &firstCompressed[0] {
		t.Error("第二次写入应该复用 compressed 缓存")
	}

	result3 := frame.WriteTo(conn3)
	if result3 != true {
		t.Error("向第三个连接写入应该成功")
	}
}

// TestFrame_ConcurrentAccess 测试并发访问
func TestFrame_ConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn := createTestWsConn(uint64(id))
			data := []byte("concurrent test message")
			frame := FrameObjPool.Get(ws.OpText, data)
			defer FrameObjPool.Put(frame)

			result := frame.WriteTo(conn)
			if result != true {
				t.Errorf("goroutine %d: 期望写入成功", id)
			}
		}(i)
	}

	wg.Wait()
}

// TestFrame_MultipleWritesCacheReuse 测试多次写入时的缓存复用
func TestFrame_MultipleWritesCacheReuse(t *testing.T) {
	conns := make([]*wsServer.Conn, 5)
	for i := range conns {
		conns[i] = createTestWsConn(uint64(i))
	}

	data := []byte("reuse test message")
	frame := FrameObjPool.Get(ws.OpText, data)
	defer FrameObjPool.Put(frame)

	result1 := frame.WriteTo(conns[0])
	if !result1 {
		t.Fatal("第一次写入应该成功")
	}

	cacheAfterFirst := frame.uncompressed
	if cacheAfterFirst == nil {
		t.Fatal("第一次写入后应该有缓存")
	}

	for i := 1; i < len(conns); i++ {
		result := frame.WriteTo(conns[i])
		if !result {
			t.Errorf("第 %d 次写入应该成功", i+1)
		}

		if &frame.uncompressed[0] != &cacheAfterFirst[0] {
			t.Errorf("第 %d 次写入应该复用缓存", i+1)
		}
	}
}

// BenchmarkFrame_GetAndPut 基准测试：Get 和 Put 的性能
func BenchmarkFrame_GetAndPut(b *testing.B) {
	data := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame := FrameObjPool.Get(ws.OpText, data)
		FrameObjPool.Put(frame)
	}
}

// BenchmarkFrame_WriteTo 基准测试：WriteTo 的性能
func BenchmarkFrame_WriteTo(b *testing.B) {
	conn := createTestWsConn(1)
	data := []byte("benchmark write message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame := FrameObjPool.Get(ws.OpText, data)
		frame.WriteTo(conn)
		FrameObjPool.Put(frame)
	}
}

// BenchmarkFrame_WriteToMultipleConns 基准测试：向多个连接写入的性能
func BenchmarkFrame_WriteToMultipleConns(b *testing.B) {
	conns := make([]*wsServer.Conn, 10)
	for i := range conns {
		conns[i] = createTestWsConn(uint64(i))
	}

	data := []byte("benchmark broadcast message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame := FrameObjPool.Get(ws.OpText, data)
		for _, conn := range conns {
			frame.WriteTo(conn)
		}
		FrameObjPool.Put(frame)
	}
}
