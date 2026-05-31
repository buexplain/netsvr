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

// TestBinder_BasicOperations 测试基本操作
func TestBinder_BasicOperations(t *testing.T) {
	binder := &collect{}
	for i := range binder.shards {
		binder.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 初始状态
	if binder.Len() != 0 {
		t.Errorf("初始长度期望=0, 实际=%d", binder.Len())
	}

	ids := binder.GetCustomerIds()
	if ids != nil {
		t.Errorf("空状态期望返回nil, 实际=%v", ids)
	}

	conns := binder.GetConnListByCustomerId("test")
	if conns != nil {
		t.Errorf("不存在的客户期望返回nil, 实际=%v", conns)
	}
}

// TestBinder_SetAndDelRelation 测试设置和删除关系
func TestBinder_SetAndDelRelation(t *testing.T) {
	binder := &collect{}
	for i := range binder.shards {
		binder.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 手动创建测试数据
	customerId := "cust1"
	connId1 := uint64(1)
	connId2 := uint64(2)

	idx := hashCustomerId(customerId)
	sd := &binder.shards[idx]

	// 设置第一个连接
	sd.mux.Lock()
	if _, ok := sd.data[customerId]; !ok {
		sd.data[customerId] = make(map[uint64]*wsServer.Conn)
	}
	sd.data[customerId][connId1] = nil
	sd.mux.Unlock()

	// 验证
	if binder.Len() != 1 {
		t.Errorf("添加后期望客户数=1, 实际=%d", binder.Len())
	}

	conns := binder.GetConnListByCustomerId(customerId)
	if len(conns) != 1 {
		t.Errorf("期望连接数=1, 实际=%d", len(conns))
	}

	// 设置第二个连接
	sd.mux.Lock()
	sd.data[customerId][connId2] = nil
	sd.mux.Unlock()

	conns = binder.GetConnListByCustomerId(customerId)
	if len(conns) != 2 {
		t.Errorf("期望连接数=2, 实际=%d", len(conns))
	}

	// 删除一个连接
	sd.mux.Lock()
	delete(sd.data[customerId], connId1)
	if len(sd.data[customerId]) == 0 {
		delete(sd.data, customerId)
	}
	sd.mux.Unlock()

	conns = binder.GetConnListByCustomerId(customerId)
	if len(conns) != 1 {
		t.Errorf("删除后期望连接数=1, 实际=%d", len(conns))
	}

	// 删除最后一个连接
	sd.mux.Lock()
	delete(sd.data[customerId], connId2)
	if len(sd.data[customerId]) == 0 {
		delete(sd.data, customerId)
	}
	sd.mux.Unlock()

	if binder.Len() != 0 {
		t.Errorf("删除所有后期望客户数=0, 实际=%d", binder.Len())
	}
}

// TestBinder_GetConnListByCustomerIds 测试获取多个客户的连接列表
func TestBinder_GetConnListByCustomerIds(t *testing.T) {
	binder := &collect{}
	for i := range binder.shards {
		binder.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 手动创建测试数据
	customerIds := []string{"cust1", "cust2"}
	for _, cid := range customerIds {
		idx := hashCustomerId(cid)
		sd := &binder.shards[idx]
		sd.mux.Lock()
		sd.data[cid] = make(map[uint64]*wsServer.Conn)
		sd.data[cid][uint64(1)] = nil
		sd.mux.Unlock()
	}

	result := binder.GetConnListByCustomerIds(customerIds)
	if len(result) != 2 {
		t.Errorf("期望客户数=2, 实际=%d", len(result))
	}

	// 测试包含空字符串
	result2 := binder.GetConnListByCustomerIds([]string{"cust1", "", "cust2"})
	if len(result2) != 2 {
		t.Errorf("包含空字符串时期望客户数=2, 实际=%d", len(result2))
	}
}

// TestBinder_ConcurrentAccess 测试并发安全性
func TestBinder_ConcurrentAccess(t *testing.T) {
	binder := &collect{}
	for i := range binder.shards {
		binder.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	var wg sync.WaitGroup
	concurrentOps := 100

	// 并发写入不同的shard
	for i := 0; i < concurrentOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn := createTestConn(uint64(id))
			customerId := string(rune('A' + id%26))
			binder.SetRelation(customerId, conn)
		}(i)
	}

	wg.Wait()

	// 验证没有panic且数据正确
	totalCustomers := binder.Len()
	if totalCustomers == 0 {
		t.Error("并发写入后客户数不应为0")
	}
}

// TestBinder_HashDistribution 测试哈希分布
func TestBinder_HashDistribution(t *testing.T) {
	shardCounts := make([]int, shardCount)
	testCustomerIds := []string{
		"customer1", "customer2", "customer3", "test_user",
		"user_123", "abc", "xyz", "hello_world",
		"foo", "bar", "baz", "qux",
	}

	for _, customerId := range testCustomerIds {
		idx := hashCustomerId(customerId)
		if idx < 0 || idx >= shardCount {
			t.Errorf("hashCustomerId(%s) = %d, 超出范围 [0, %d)", customerId, idx, shardCount)
		}
		shardCounts[idx]++
	}

	t.Logf("哈希分布情况: %+v", shardCounts)
}

// TestBinder_DelRelation 测试删除关系方法
func TestBinder_DelRelation(t *testing.T) {
	binder := &collect{}
	for i := range binder.shards {
		binder.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	customerId := "cust1"
	conn := createTestConn(1)

	// 设置关系
	binder.SetRelation(customerId, conn)
	if binder.Len() != 1 {
		t.Fatalf("添加后期望客户数=1, 实际=%d", binder.Len())
	}

	// 删除关系
	binder.DelRelation(customerId, conn)
	if binder.Len() != 0 {
		t.Errorf("删除后期望客户数=0, 实际=%d", binder.Len())
	}

	// 验证连接列表为空
	conns := binder.GetConnListByCustomerId(customerId)
	if conns != nil {
		t.Errorf("删除后期望返回nil, 实际=%v", conns)
	}
}

// TestBinder_GetCustomerIds 测试获取所有客户ID
func TestBinder_GetCustomerIds(t *testing.T) {
	binder := &collect{}
	for i := range binder.shards {
		binder.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 空状态
	ids := binder.GetCustomerIds()
	if ids != nil {
		t.Errorf("空状态期望返回nil, 实际=%v", ids)
	}

	// 添加多个客户
	for i := 0; i < 5; i++ {
		conn := createTestConn(uint64(i))
		customerId := string(rune('A' + i))
		binder.SetRelation(customerId, conn)
	}

	ids = binder.GetCustomerIds()
	if len(ids) != 5 {
		t.Errorf("期望客户数=5, 实际=%d", len(ids))
	}
}

// BenchmarkBinder_Operations 性能基准测试
func BenchmarkBinder_Operations(b *testing.B) {
	binder := &collect{}
	for i := range binder.shards {
		binder.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := createTestConn(uint64(i))
		customerId := string(rune('A' + i%26))
		binder.SetRelation(customerId, conn)
	}
}
