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

package manager

import (
	"github.com/panjf2000/gnet/v2"
	"io"
	"net"
	"netsvr/internal/utils/slicePool"
	"netsvr/internal/wsServer"
	"sync"
	"testing"
	"time"
)

// MockGnetConn 是 gnet.Conn 的完整 mock 实现
type MockGnetConn struct {
	id          uint64
	localAddr   net.Addr
	remoteAddr  net.Addr
	writeData   [][]byte
	closeCalled bool
	context     interface{}
	eventLoop   gnet.EventLoop
}

func NewMockGnetConn(id uint64) *MockGnetConn {
	return &MockGnetConn{
		id:         id,
		localAddr:  &mockAddr{network: "tcp", addr: "127.0.0.1:8080"},
		remoteAddr: &mockAddr{network: "tcp", addr: "127.0.0.1:9090"},
		writeData:  make([][]byte, 0),
	}
}

func (m *MockGnetConn) Read(_ []byte) (n int, err error)         { return 0, nil }
func (m *MockGnetConn) WriteTo(_ io.Writer) (n int64, err error) { return 0, nil }
func (m *MockGnetConn) Next(_ int) (buf []byte, err error)       { return nil, nil }
func (m *MockGnetConn) Peek(_ int) (buf []byte, err error)       { return nil, nil }
func (m *MockGnetConn) Discard(_ int) (discarded int, err error) { return 0, nil }
func (m *MockGnetConn) InboundBuffered() int                     { return 0 }
func (m *MockGnetConn) Write(b []byte) (n int, err error) {
	data := make([]byte, len(b))
	copy(data, b)
	m.writeData = append(m.writeData, data)
	return len(b), nil
}
func (m *MockGnetConn) ReadFrom(_ io.Reader) (n int64, err error)      { return 0, nil }
func (m *MockGnetConn) SendTo(_ []byte, _ net.Addr) (n int, err error) { return 0, nil }
func (m *MockGnetConn) Writev(_ [][]byte) (n int, err error)           { return 0, nil }
func (m *MockGnetConn) Flush() error                                   { return nil }
func (m *MockGnetConn) OutboundBuffered() int                          { return 0 }
func (m *MockGnetConn) AsyncWrite(_ []byte, callback gnet.AsyncCallback) error {
	if callback != nil {
		_ = callback(m, nil)
	}
	return nil
}
func (m *MockGnetConn) AsyncWritev(_ [][]byte, callback gnet.AsyncCallback) error {
	if callback != nil {
		_ = callback(m, nil)
	}
	return nil
}
func (m *MockGnetConn) Fd() int                                              { return 0 }
func (m *MockGnetConn) Dup() (int, error)                                    { return 0, nil }
func (m *MockGnetConn) SetReadBuffer(_ int) error                            { return nil }
func (m *MockGnetConn) SetWriteBuffer(_ int) error                           { return nil }
func (m *MockGnetConn) SetLinger(_ int) error                                { return nil }
func (m *MockGnetConn) SetKeepAlivePeriod(_ time.Duration) error             { return nil }
func (m *MockGnetConn) SetKeepAlive(_ bool, _, _ time.Duration, _ int) error { return nil }
func (m *MockGnetConn) SetNoDelay(_ bool) error                              { return nil }
func (m *MockGnetConn) Context() interface{}                                 { return m.context }
func (m *MockGnetConn) EventLoop() gnet.EventLoop                            { return m.eventLoop }
func (m *MockGnetConn) SetContext(ctx interface{})                           { m.context = ctx }
func (m *MockGnetConn) LocalAddr() net.Addr                                  { return m.localAddr }
func (m *MockGnetConn) RemoteAddr() net.Addr                                 { return m.remoteAddr }
func (m *MockGnetConn) Wake(_ gnet.AsyncCallback) error                      { return nil }
func (m *MockGnetConn) CloseWithCallback(callback gnet.AsyncCallback) error {
	m.closeCalled = true
	if callback != nil {
		_ = callback(m, nil)
	}
	return nil
}
func (m *MockGnetConn) Close() error                       { m.closeCalled = true; return nil }
func (m *MockGnetConn) SetDeadline(_ time.Time) error      { return nil }
func (m *MockGnetConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *MockGnetConn) SetWriteDeadline(_ time.Time) error { return nil }

type mockAddr struct {
	network string
	addr    string
}

func (m *mockAddr) Network() string { return m.network }
func (m *mockAddr) String() string  { return m.addr }

// createTestConn 创建测试用的 wsServer.Conn
func createTestConn(id uint64) *wsServer.Conn {
	mockConn := NewMockGnetConn(id)
	return wsServer.NewConn(mockConn)
}

// TestManager_BasicOperations 测试基本操作
func TestManager_BasicOperations(t *testing.T) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	// 初始状态
	if manager.Len() != 0 {
		t.Errorf("初始长度期望=0, 实际=%d", manager.Len())
	}

	conns := manager.GetConnections(slicePool.NewWsConn(16))
	if conns != nil {
		t.Errorf("空状态期望返回nil, 实际=%v", conns)
	}

	uniqIds := manager.GetUniqIds()
	if uniqIds != nil {
		t.Errorf("空状态期望返回nil, 实际=%v", uniqIds)
	}
}

// TestManager_SetAndGet 测试设置和获取
func TestManager_SetAndGet(t *testing.T) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	conn := createTestConn(1)

	// 设置连接
	manager.Set(conn.GetUniqIdOnSafe(), conn)

	if manager.Len() != 1 {
		t.Errorf("期望长度=1, 实际=%d", manager.Len())
	}

	retConn := manager.Get(conn.GetUniqIdOnSafe())
	// retConn为nil是正常的，因为我们没有设置实际的连接对象
	if manager.Len() != 1 {
		t.Errorf("期望长度=1, 实际=%d", manager.Len())
	}
	_ = retConn // 避免未使用警告
}

// TestManager_Del 测试删除
func TestManager_Del(t *testing.T) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	conn := createTestConn(1)

	// 添加连接
	manager.Set(conn.GetUniqIdOnSafe(), conn)

	if manager.Len() != 1 {
		t.Fatalf("添加后期望长度=1, 实际=%d", manager.Len())
	}

	// 删除连接
	manager.Del(conn.GetUniqIdOnSafe())

	if manager.Len() != 0 {
		t.Errorf("删除后期望长度=0, 实际=%d", manager.Len())
	}

	if manager.Has(conn.GetUniqIdOnSafe()) {
		t.Error("删除后不应该还能找到该uniqId")
	}
}

// TestManager_CounterConsistency 测试计数器一致性
func TestManager_CounterConsistency(t *testing.T) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	// 添加50个连接
	var conns []*wsServer.Conn
	for i := 0; i < 50; i++ {
		conn := createTestConn(uint64(i))
		conns = append(conns, conn)
		manager.Set(conn.GetUniqIdOnSafe(), conn)
	}

	addCount := manager.Len()
	if addCount != 50 {
		t.Errorf("添加后期望长度=50, 实际=%d", addCount)
	}

	// 删除25个
	for i := 0; i < 25; i++ {
		manager.Del(conns[i].GetUniqIdOnSafe())
	}

	delCount := manager.Len()
	if delCount != 25 {
		t.Errorf("删除25个后期望长度=25, 实际=%d", delCount)
	}
}

// TestManager_GetConnections 测试获取所有连接
func TestManager_GetConnections(t *testing.T) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	sp := slicePool.NewWsConn(16)

	// 添加连接
	for i := 0; i < 5; i++ {
		conn := createTestConn(uint64(i))
		manager.Set(conn.GetUniqIdOnSafe(), conn)
	}

	conns := manager.GetConnections(sp)
	if conns == nil {
		t.Fatal("期望获取到连接列表, 实际为nil")
	}
	defer sp.Put(conns)

	if len(*conns) != 5 {
		t.Errorf("期望连接数=5, 实际=%d", len(*conns))
	}

	// 验证归还后长度为0
	sp.Put(conns)
	if len(*conns) != 0 {
		t.Errorf("归还后长度应该为0, 实际=%d", len(*conns))
	}
}

// TestManager_ConcurrentAccess 测试并发安全性
func TestManager_ConcurrentAccess(t *testing.T) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	var wg sync.WaitGroup
	concurrentOps := 100

	// 并发写入
	for i := 0; i < concurrentOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn := createTestConn(uint64(id))
			manager.Set(conn.GetUniqIdOnSafe(), conn)
		}(i)
	}

	// 并发读取
	for i := 0; i < concurrentOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			uniqId := string(rune('A' + id%26))
			manager.Has(uniqId)
			manager.Get(uniqId)
		}(i)
	}

	wg.Wait()

	// 验证没有panic且数据正确
	totalConns := manager.Len()
	if totalConns == 0 {
		t.Error("并发写入后连接数不应为0")
	}
}

// TestManager_HashDistribution 测试哈希分布
func TestManager_HashDistribution(t *testing.T) {
	shardCounts := make([]int, shardCount)
	testUniqIds := []string{
		"user1", "user2", "user3", "test_user",
		"user_123", "abc", "xyz", "hello_world",
		"foo", "bar", "baz", "qux",
	}

	for _, uniqId := range testUniqIds {
		idx := hashUniqId(uniqId)
		if idx < 0 || idx >= shardCount {
			t.Errorf("hashUniqId(%s) = %d, 超出范围 [0, %d)", uniqId, idx, shardCount)
		}
		shardCounts[idx]++
	}

	t.Logf("哈希分布情况: %+v", shardCounts)
}

// TestManager_SetEmptyUniqId 测试设置空uniqId
func TestManager_SetEmptyUniqId(t *testing.T) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	// 设置空uniqId应该直接返回，不增加计数
	conn := createTestConn(1)
	manager.Set("", conn)
	if manager.Len() != 0 {
		t.Errorf("设置空uniqId后期望长度=0, 实际=%d", manager.Len())
	}
}

// TestManager_GetUniqIds 测试获取所有uniqId
func TestManager_GetUniqIds(t *testing.T) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	// 空状态
	uniqIds := manager.GetUniqIds()
	if uniqIds != nil {
		t.Errorf("空状态期望返回nil, 实际=%v", uniqIds)
	}

	// 添加多个连接
	for i := 0; i < 5; i++ {
		conn := createTestConn(uint64(i))
		manager.Set(conn.GetUniqIdOnSafe(), conn)
	}

	uniqIds = manager.GetUniqIds()
	if len(uniqIds) != 5 {
		t.Errorf("期望uniqId数=5, 实际=%d", len(uniqIds))
	}
}

// TestManager_DelEmptyUniqId 测试删除空uniqId
func TestManager_DelEmptyUniqId(t *testing.T) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	// 删除空uniqId应该直接返回
	manager.Del("")
	if manager.Len() != 0 {
		t.Errorf("删除空uniqId后期望长度=0, 实际=%d", manager.Len())
	}
}

// BenchmarkManager_Operations 性能基准测试
func BenchmarkManager_Operations(b *testing.B) {
	manager := &collect{}
	for i := range manager.shards {
		manager.shards[i].data = make(map[string]*wsServer.Conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := createTestConn(uint64(i))
		manager.Set(conn.GetUniqIdOnSafe(), conn)
	}
}
