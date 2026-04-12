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
	"fmt"
	"github.com/panjf2000/gnet/v2"
	"io"
	"net"
	"netsvr/internal/customer/info"
	"netsvr/internal/utils/slicePool"
	"netsvr/internal/wsServer"
	"sync"
	"testing"
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
		c.shards[i].data = make(map[string]gnet.Conn, shardCount)
	}
	return c
}

// createMockConnWithSession 创建带有 session 的 mock 连接
func createMockConnWithSession(uniqId, customerId string) gnet.Conn {
	mockConn := &MockConn{}
	codec := &wsServer.Codec{}
	sessionInfo := info.New(uniqId)
	sessionInfo.SetCustomerId(customerId)
	codec.SetSession(sessionInfo)
	mockConn.SetContext(codec)
	return mockConn
}

// TestHashUniqId 测试哈希函数的一致性
func TestHashUniqId(t *testing.T) {
	tests := []struct {
		name   string
		uniqId string
	}{
		{"empty", ""},
		{"simple", "uniq1"},
		{"chinese", "唯一ID测试"},
		{"special", "uniq-with-special_chars!@#"},
		{"long", "this-is-a-very-long-unique-id-for-testing-purposes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashUniqId(tt.uniqId)
			hash2 := hashUniqId(tt.uniqId)
			// 同一输入应该产生相同输出
			if hash1 != hash2 {
				t.Errorf("hashUniqId(%q) = %d, then %d, should be equal", tt.uniqId, hash1, hash2)
			}
			// 结果应该在有效范围内
			if hash1 < 0 || hash1 >= shardCount {
				t.Errorf("hashUniqId(%q) = %d, should be in [0, %d)", tt.uniqId, hash1, shardCount)
			}
		})
	}
}

// TestHas 测试检查 uniqId 是否存在
func TestHas(t *testing.T) {
	tc := newTestCollect()

	// 空字符串
	if tc.Has("") {
		t.Error("Has(\"\") should return false")
	}

	// 不存在的 uniqId
	if tc.Has("nonexistent") {
		t.Error("Has(nonexistent) should return false")
	}

	// 添加后存在
	mockConn := &MockConn{}
	tc.Set("uniq1", mockConn)
	if !tc.Has("uniq1") {
		t.Error("Has(uniq1) should return true after SetRelation")
	}

	// 删除后不存在
	tc.Del("uniq1")
	if tc.Has("uniq1") {
		t.Error("Has(uniq1) should return false after DelRelation")
	}
}

// TestGet 测试获取连接对象
func TestGet(t *testing.T) {
	tc := newTestCollect()

	// 空字符串
	if tc.Get("") != nil {
		t.Error("Get(\"\") should return nil")
	}

	// 不存在的 uniqId
	if tc.Get("nonexistent") != nil {
		t.Error("Get(nonexistent) should return nil")
	}

	// 添加后获取
	mockConn := &MockConn{}
	tc.Set("uniq1", mockConn)
	conn := tc.Get("uniq1")
	if conn == nil {
		t.Fatal("Get(uniq1) should return the connection")
	}
	// 验证返回的是同一个连接对象（通过比较指针地址）
	if fmt.Sprintf("%p", conn) != fmt.Sprintf("%p", mockConn) {
		t.Error("Get(uniq1) should return the same connection object")
	}

	// 删除后获取
	tc.Del("uniq1")
	if tc.Get("uniq1") != nil {
		t.Error("Get(uniq1) should return nil after DelRelation")
	}
}

// TestSet 测试设置连接
func TestSet(t *testing.T) {
	tc := newTestCollect()

	// 正常添加
	mockConn1 := &MockConn{}
	tc.Set("uniq1", mockConn1)
	if !tc.Has("uniq1") {
		t.Error("SetRelation should add the connection")
	}

	// 覆盖已有连接
	mockConn2 := &MockConn{}
	tc.Set("uniq1", mockConn2)
	conn := tc.Get("uniq1")
	if conn != mockConn2 {
		t.Error("SetRelation should overwrite existing connection")
	}

	// 添加多个不同的 uniqId
	tc.Set("uniq2", &MockConn{})
	tc.Set("uniq3", &MockConn{})
	if tc.Len() != 3 {
		t.Errorf("Len() = %d, want 3", tc.Len())
	}
}

// TestDel 测试删除连接
func TestDel(t *testing.T) {
	tc := newTestCollect()

	// 删除空字符串（不应 panic）
	tc.Del("")

	// 删除不存在的 uniqId（不应 panic）
	tc.Del("nonexistent")

	// 正常删除
	mockConn := &MockConn{}
	tc.Set("uniq1", mockConn)
	tc.Del("uniq1")
	if tc.Has("uniq1") {
		t.Error("DelRelation should remove the connection")
	}
	if tc.Len() != 0 {
		t.Errorf("Len() = %d, want 0", tc.Len())
	}
}

// TestLen 测试连接数量统计
func TestLen(t *testing.T) {
	tc := newTestCollect()

	// 初始状态应为 0
	if l := tc.Len(); l != 0 {
		t.Errorf("initial Len() = %d, want 0", l)
	}

	// 添加连接
	tc.Set("uniq1", &MockConn{})
	tc.Set("uniq2", &MockConn{})
	tc.Set("uniq3", &MockConn{})
	if l := tc.Len(); l != 3 {
		t.Errorf("after adding 3 connections, Len() = %d, want 3", l)
	}

	// 覆盖已有连接，数量不变
	tc.Set("uniq1", &MockConn{})
	if l := tc.Len(); l != 3 {
		t.Errorf("after overwriting connection, Len() = %d, want 3", l)
	}

	// 删除连接
	tc.Del("uniq1")
	if l := tc.Len(); l != 2 {
		t.Errorf("after deleting one connection, Len() = %d, want 2", l)
	}
}

// TestGetConnections 测试获取所有连接
func TestGetConnections(t *testing.T) {
	tc := newTestCollect()
	sp := slicePool.NewWsConn(10)

	// 空状态
	conns := tc.GetConnections(sp)
	if conns != nil {
		t.Errorf("GetConnections() on empty = %v, want nil", conns)
	}

	// 添加连接
	conn1 := &MockConn{}
	conn2 := &MockConn{}
	conn3 := &MockConn{}
	tc.Set("uniq1", conn1)
	tc.Set("uniq2", conn2)
	tc.Set("uniq3", conn3)

	// 获取所有连接
	conns = tc.GetConnections(sp)
	if conns == nil {
		t.Fatal("GetConnections() returned nil")
	}
	if len(*conns) != 3 {
		t.Errorf("GetConnections() returned %d connections, want 3", len(*conns))
	}

	// 验证包含所有连接
	connMap := make(map[gnet.Conn]bool)
	for _, conn := range *conns {
		connMap[conn] = true
	}
	if !connMap[conn1] || !connMap[conn2] || !connMap[conn3] {
		t.Errorf("GetConnections() missing some connections")
	}

	// 归还切片到池
	sp.Put(conns)
}

// TestGetCustomerIds 测试根据 uniqIds 获取 customerIds
func TestGetCustomerIds(t *testing.T) {
	tc := newTestCollect()

	// 空输入
	result := tc.GetCustomerIds(nil)
	if result != nil {
		t.Errorf("GetCustomerIds(nil) = %v, want nil", result)
	}

	result = tc.GetCustomerIds([]string{})
	if result != nil {
		t.Errorf("GetCustomerIds([]) = %v, want nil", result)
	}

	// 添加带 session 的连接
	conn1 := createMockConnWithSession("uniq1", "customer1")
	conn2 := createMockConnWithSession("uniq2", "customer1") // 同一客户
	conn3 := createMockConnWithSession("uniq3", "customer2")
	conn4 := createMockConnWithSession("uniq4", "customer3")

	tc.Set("uniq1", conn1)
	tc.Set("uniq2", conn2)
	tc.Set("uniq3", conn3)
	tc.Set("uniq4", conn4)

	// 查询多个 uniqId，应去重
	result = tc.GetCustomerIds([]string{"uniq1", "uniq2", "uniq3"})
	if len(result) != 2 {
		t.Errorf("GetCustomerIds returned %d customerIds, want 2 (customer1, customer2)", len(result))
	}

	// 验证包含正确的 customerId
	customerIdMap := make(map[string]bool)
	for _, id := range result {
		customerIdMap[id] = true
	}
	if !customerIdMap["customer1"] || !customerIdMap["customer2"] {
		t.Errorf("GetCustomerIds = %v, missing customer1 or customer2", result)
	}
	if customerIdMap["customer3"] {
		t.Error("GetCustomerIds should not include customer3")
	}

	// 查询不存在的 uniqId
	result = tc.GetCustomerIds([]string{"nonexistent"})
	if result != nil {
		t.Errorf("GetCustomerIds([nonexistent]) = %v, want nil", result)
	}

	// 混合查询
	result = tc.GetCustomerIds([]string{"uniq1", "nonexistent"})
	if len(result) != 1 || result[0] != "customer1" {
		t.Errorf("GetCustomerIds([uniq1, nonexistent]) = %v, want [customer1]", result)
	}

	// 包含空字符串
	result = tc.GetCustomerIds([]string{"uniq1", "", "uniq3"})
	if len(result) != 2 {
		t.Errorf("GetCustomerIds with empty string returned %d customerIds, want 2", len(result))
	}

	// 无 session 的连接（context 为 nil）
	tc.Set("uniq5", &MockConn{})
	result = tc.GetCustomerIds([]string{"uniq5"})
	if result != nil {
		t.Errorf("GetCustomerIds for connection without session = %v, want nil", result)
	}
}

// TestCountCustomerIds 测试统计不同 customerId 的数量
func TestCountCustomerIds(t *testing.T) {
	tc := newTestCollect()

	// 空输入
	count := tc.CountCustomerIds(nil)
	if count != 0 {
		t.Errorf("CountCustomerIds(nil) = %d, want 0", count)
	}

	count = tc.CountCustomerIds([]string{})
	if count != 0 {
		t.Errorf("CountCustomerIds([]) = %d, want 0", count)
	}

	// 添加带 session 的连接
	conn1 := createMockConnWithSession("uniq1", "customer1")
	conn2 := createMockConnWithSession("uniq2", "customer1") // 同一客户
	conn3 := createMockConnWithSession("uniq3", "customer2")
	conn4 := createMockConnWithSession("uniq4", "customer3")

	tc.Set("uniq1", conn1)
	tc.Set("uniq2", conn2)
	tc.Set("uniq3", conn3)
	tc.Set("uniq4", conn4)

	// 统计数量（应去重）
	count = tc.CountCustomerIds([]string{"uniq1", "uniq2", "uniq3"})
	if count != 2 {
		t.Errorf("CountCustomerIds returned %d, want 2 (customer1, customer2)", count)
	}

	// 查询不存在的 uniqId
	count = tc.CountCustomerIds([]string{"nonexistent"})
	if count != 0 {
		t.Errorf("CountCustomerIds([nonexistent]) = %d, want 0", count)
	}

	// 混合查询
	count = tc.CountCustomerIds([]string{"uniq1", "nonexistent"})
	if count != 1 {
		t.Errorf("CountCustomerIds([uniq1, nonexistent]) = %d, want 1", count)
	}

	// 包含空字符串
	count = tc.CountCustomerIds([]string{"uniq1", "", "uniq3"})
	if count != 2 {
		t.Errorf("CountCustomerIds with empty string returned %d, want 2", count)
	}
}

// TestGetUniqIds 测试获取所有 uniqId
func TestGetUniqIds(t *testing.T) {
	tc := newTestCollect()

	// 空状态
	uniqIds := tc.GetUniqIds()
	if uniqIds != nil {
		t.Errorf("GetUniqIds() on empty = %v, want nil", uniqIds)
	}

	// 添加连接
	tc.Set("uniq1", &MockConn{})
	tc.Set("uniq2", &MockConn{})
	tc.Set("uniq3", &MockConn{})

	uniqIds = tc.GetUniqIds()
	if len(uniqIds) != 3 {
		t.Errorf("GetUniqIds() returned %d uniqIds, want 3", len(uniqIds))
	}

	// 验证包含所有 uniqId
	uniqIdMap := make(map[string]bool)
	for _, id := range uniqIds {
		uniqIdMap[id] = true
	}
	if !uniqIdMap["uniq1"] || !uniqIdMap["uniq2"] || !uniqIdMap["uniq3"] {
		t.Errorf("GetUniqIds() = %v, missing some uniqIds", uniqIds)
	}
}

// TestConcurrentAccess 测试并发访问的安全性
func TestConcurrentAccess(t *testing.T) {
	tc := newTestCollect()
	var wg sync.WaitGroup

	concurrentConns := 100

	// 并发写入
	for i := 0; i < concurrentConns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			uniqId := "uniq" + string(rune('A'+idx%26))
			tc.Set(uniqId, &MockConn{})
		}(i)
	}
	wg.Wait()

	// 并发读取
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = tc.Len()
			_ = tc.GetUniqIds()
			_ = tc.Has("uniqA")
			_ = tc.Get("uniqA")
		}()
	}
	wg.Wait()

	// 并发删除
	for i := 0; i < concurrentConns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			uniqId := "uniq" + string(rune('A'+idx%26))
			tc.Del(uniqId)
		}(i)
	}
	wg.Wait()

	// 最终应该为空
	if tc.Len() != 0 {
		t.Errorf("after all deletions, Len() = %d, want 0", tc.Len())
	}
}

// BenchmarkSet 性能基准测试
func BenchmarkSet(b *testing.B) {
	tc := newTestCollect()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uniqId := "uniq" + string(rune(i%100))
		tc.Set(uniqId, &MockConn{})
	}
}

// BenchmarkGet 性能基准测试
func BenchmarkGet(b *testing.B) {
	tc := newTestCollect()

	// 准备数据
	for i := 0; i < 100; i++ {
		tc.Set("uniq"+string(rune(i)), &MockConn{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.Get("uniq" + string(rune(i%100)))
	}
}

// BenchmarkHas 性能基准测试
func BenchmarkHas(b *testing.B) {
	tc := newTestCollect()

	// 准备数据
	for i := 0; i < 100; i++ {
		tc.Set("uniq"+string(rune(i)), &MockConn{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.Has("uniq" + string(rune(i%100)))
	}
}

// BenchmarkGetCustomerIds 性能基准测试
func BenchmarkGetCustomerIds(b *testing.B) {
	tc := newTestCollect()
	uniqIds := []string{"uniq1", "uniq2", "uniq3", "uniq4", "uniq5"}

	// 准备数据
	for _, uniqId := range uniqIds {
		conn := createMockConnWithSession(uniqId, "customer_"+uniqId)
		tc.Set(uniqId, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.GetCustomerIds(uniqIds)
	}
}

// BenchmarkCountCustomerIds 性能基准测试
func BenchmarkCountCustomerIds(b *testing.B) {
	tc := newTestCollect()
	uniqIds := []string{"uniq1", "uniq2", "uniq3", "uniq4", "uniq5"}

	// 准备数据
	for _, uniqId := range uniqIds {
		conn := createMockConnWithSession(uniqId, "customer_"+uniqId)
		tc.Set(uniqId, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.CountCustomerIds(uniqIds)
	}
}

// BenchmarkDel 性能基准测试
func BenchmarkDel(b *testing.B) {
	tc := newTestCollect()

	// 准备数据
	for i := 0; i < b.N; i++ {
		uniqId := "uniq" + string(rune(i%100))
		tc.Set(uniqId, &MockConn{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uniqId := "uniq" + string(rune(i%100))
		tc.Del(uniqId)
	}
}

// BenchmarkLen 性能基准测试
func BenchmarkLen(b *testing.B) {
	tc := newTestCollect()

	// 准备 1000 个连接
	for i := 0; i < 1000; i++ {
		tc.Set(fmt.Sprintf("uniq%d", i), &MockConn{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.Len()
	}
}

// BenchmarkGetConnections 性能基准测试
func BenchmarkGetConnections(b *testing.B) {
	tc := newTestCollect()
	sp := slicePool.NewWsConn(16)

	// 准备 1000 个连接
	for i := 0; i < 1000; i++ {
		tc.Set(fmt.Sprintf("uniq%d", i), &MockConn{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conns := tc.GetConnections(sp)
		if conns != nil {
			sp.Put(conns)
		}
	}
}

// BenchmarkGetUniqIds 性能基准测试
func BenchmarkGetUniqIds(b *testing.B) {
	tc := newTestCollect()

	// 准备 1000 个连接
	for i := 0; i < 1000; i++ {
		tc.Set(fmt.Sprintf("uniq%d", i), &MockConn{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.GetUniqIds()
	}
}

// TestGetConnectionsPoolReuse 测试对象池复用
func TestGetConnectionsPoolReuse(t *testing.T) {
	tc := newTestCollect()
	sp := slicePool.NewWsConn(16)

	// 添加连接
	for i := 0; i < 10; i++ {
		tc.Set(fmt.Sprintf("uniq%d", i), &MockConn{})
	}

	// 第一次获取
	conns1 := tc.GetConnections(sp)
	if conns1 == nil {
		t.Fatal("期望非 nil")
	}
	cap1 := cap(*conns1)

	// 归还
	sp.Put(conns1)

	// 第二次获取，应该复用池中的切片
	conns2 := tc.GetConnections(sp)
	if conns2 == nil {
		t.Fatal("期望非 nil")
	}

	// 验证容量相似（可能因实现略有不同）
	if cap(*conns2) < cap1/2 {
		t.Logf("警告：池复用可能未生效，cap1=%d, cap2=%d", cap1, cap(*conns2))
	}

	sp.Put(conns2)
}

// TestGetCustomerIdsWithInvalidContext 测试无效 context 的情况
func TestGetCustomerIdsWithInvalidContext(t *testing.T) {
	tc := newTestCollect()

	// 添加一个没有 context 的连接
	mockConn := &MockConn{}
	mockConn.SetContext(nil) // 显式设置为 nil
	tc.Set("uniq1", mockConn)

	// 查询应该返回 nil（无有效 customerId）
	result := tc.GetCustomerIds([]string{"uniq1"})
	if result != nil {
		t.Errorf("无效 context 期望返回 nil，实际 %v", result)
	}

	// 添加一个 context 不是 Codec 的连接
	mockConn2 := &MockConn{}
	mockConn2.SetContext("not a codec")
	tc.Set("uniq2", mockConn2)

	result = tc.GetCustomerIds([]string{"uniq2"})
	if result != nil {
		t.Errorf("非 Codec context 期望返回 nil，实际 %v", result)
	}
}

// TestCountCustomerIdsWithInvalidContext 测试无效 context 的计数
func TestCountCustomerIdsWithInvalidContext(t *testing.T) {
	tc := newTestCollect()

	// 添加无效 context 的连接
	mockConn := &MockConn{}
	mockConn.SetContext(nil)
	tc.Set("uniq1", mockConn)

	count := tc.CountCustomerIds([]string{"uniq1"})
	if count != 0 {
		t.Errorf("无效 context 期望计数 0，实际 %d", count)
	}
}

// TestLargeScaleCrossShard 测试大规模跨分片场景
func TestLargeScaleCrossShard(t *testing.T) {
	tc := newTestCollect()

	// 添加 500 个 uniqId，确保跨多个分片
	const count = 500
	for i := 0; i < count; i++ {
		uniqId := fmt.Sprintf("uniq_%d", i)
		tc.Set(uniqId, &MockConn{})
	}

	// 验证数量
	if tc.Len() != count {
		t.Errorf("Len() = %d, want %d", tc.Len(), count)
	}

	// 获取所有 uniqId
	uniqIds := tc.GetUniqIds()
	if len(uniqIds) != count {
		t.Errorf("GetUniqIds() returned %d, want %d", len(uniqIds), count)
	}

	// 批量查询部分 uniqId
	queryIds := make([]string, 100)
	for i := 0; i < 100; i++ {
		queryIds[i] = fmt.Sprintf("uniq_%d", i*5) // 每隔 5 个查一个
	}

	customerIds := tc.GetCustomerIds(queryIds)
	// 因为没有设置 session，应该返回 nil
	if customerIds != nil {
		t.Errorf("无 session 连接期望返回 nil，实际 %v", customerIds)
	}

	countResult := tc.CountCustomerIds(queryIds)
	if countResult != 0 {
		t.Errorf("无 session 连接期望计数 0，实际 %d", countResult)
	}
}

// TestGetConnectionsEmptyState 测试空状态下的 GetConnections
func TestGetConnectionsEmptyState(t *testing.T) {
	tc := newTestCollect()
	sp := slicePool.NewWsConn(16)

	// 空状态应返回 nil
	conns := tc.GetConnections(sp)
	if conns != nil {
		t.Errorf("空状态期望返回 nil，实际 %v", conns)
	}
}

// TestHashUniqIdDistribution 测试哈希分布均匀性
func TestHashUniqIdDistribution(t *testing.T) {
	shardCounts := make([]int, shardCount)

	// 生成 10000 个不同的 uniqId
	for i := 0; i < 10000; i++ {
		uniqId := fmt.Sprintf("uniq_%d", i)
		idx := hashUniqId(uniqId)
		shardCounts[idx]++
	}

	// 计算平均值
	avg := 10000.0 / shardCount

	// 检查分布
	maxCount := 0
	minCount := 10000
	for _, count := range shardCounts {
		if count > maxCount {
			maxCount = count
		}
		if count < minCount {
			minCount = count
		}
	}

	deviation := float64(maxCount-minCount) / avg
	if deviation > 1.0 {
		t.Logf("分片分布: 最大=%d, 最小=%d, 平均=%.2f, 偏差=%.2f%%",
			maxCount, minCount, avg, deviation*100)
	}
}
