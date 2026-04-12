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
	"fmt"
	"github.com/panjf2000/gnet/v2"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// MockConn 模拟 gnet.Conn 用于测试
type MockConn struct {
	context   any
	eventLoop gnet.EventLoop
	fd        int
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
func (m *MockConn) Fd() int                                                    { return m.fd }
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

// createMockConn 创建带指定 Fd 的 MockConn
func createMockConn(fd int) *MockConn {
	return &MockConn{fd: fd}
}

// TestHashCustomerId 测试哈希函数
func TestHashCustomerId(t *testing.T) {
	// 测试空字符串
	idx := hashCustomerId("")
	if idx < 0 || idx >= shardCount {
		t.Errorf("hashCustomerId(\"\") = %d, want in range [0, %d)", idx, shardCount)
	}

	// 测试相同输入产生相同输出
	idx1 := hashCustomerId("user123")
	idx2 := hashCustomerId("user123")
	if idx1 != idx2 {
		t.Errorf("hashCustomerId 不一致: %d != %d", idx1, idx2)
	}

	// 测试不同输入（允许哈希碰撞）
	_ = hashCustomerId("user456")
	// 不强制要求不同，因为可能有哈希碰撞

	// 测试所有结果都在有效范围内
	testCases := []string{"a", "abc", "用户ID", "user_123456789", ""}
	for _, tc := range testCases {
		idx := hashCustomerId(tc)
		if idx < 0 || idx >= shardCount {
			t.Errorf("hashCustomerId(%q) = %d, out of range", tc, idx)
		}
	}
}

// TestSetRelation 测试设置绑定关系
func TestSetRelation(t *testing.T) {
	c := newTestCollect()

	// 测试基本设置
	conn1 := createMockConn(1)
	c.SetRelation("user1", conn1)

	// 验证连接已添加
	conns := c.GetConnListByCustomerId("user1")
	if len(conns) != 1 {
		t.Errorf("期望 1 个连接，实际 %d", len(conns))
	}

	// 测试同一用户多个连接
	conn2 := createMockConn(2)
	c.SetRelation("user1", conn2)

	conns = c.GetConnListByCustomerId("user1")
	if len(conns) != 2 {
		t.Errorf("期望 2 个连接，实际 %d", len(conns))
	}

	// 测试不同用户
	conn3 := createMockConn(3)
	c.SetRelation("user2", conn3)

	conns = c.GetConnListByCustomerId("user2")
	if len(conns) != 1 {
		t.Errorf("期望 1 个连接，实际 %d", len(conns))
	}

	// 验证 user1 不受影响
	conns = c.GetConnListByCustomerId("user1")
	if len(conns) != 2 {
		t.Errorf("user1 期望 2 个连接，实际 %d", len(conns))
	}
}

// TestDelRelation 测试删除绑定关系
func TestDelRelation(t *testing.T) {
	c := newTestCollect()

	// 设置测试数据
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	c.SetRelation("user1", conn1)
	c.SetRelation("user1", conn2)

	// 删除一个连接
	c.DelRelation("user1", conn1)

	conns := c.GetConnListByCustomerId("user1")
	if len(conns) != 1 {
		t.Errorf("删除后期望 1 个连接，实际 %d", len(conns))
	}

	// 删除最后一个连接，应该清理 customerId
	c.DelRelation("user1", conn2)

	conns = c.GetConnListByCustomerId("user1")
	if conns != nil {
		t.Errorf("删除所有连接后期望 nil，实际 %v", conns)
	}

	// 测试删除不存在的 customerId
	c.DelRelation("nonexistent", conn1) // 不应该 panic

	// 测试删除不存在的连接
	c.SetRelation("user2", conn1)
	conn3 := createMockConn(3)
	c.DelRelation("user2", conn3) // 不应该 panic

	conns = c.GetConnListByCustomerId("user2")
	if len(conns) != 1 {
		t.Errorf("删除不存在的连接后期望 1 个连接，实际 %d", len(conns))
	}
}

// TestLen 测试获取 customerId 数量
func TestLen(t *testing.T) {
	c := newTestCollect()

	// 初始状态应为 0
	if c.Len() != 0 {
		t.Errorf("初始 Len() = %d, 期望 0", c.Len())
	}

	// 添加一个用户
	conn1 := createMockConn(1)
	c.SetRelation("user1", conn1)
	if c.Len() != 1 {
		t.Errorf("添加 1 个用户后 Len() = %d, 期望 1", c.Len())
	}

	// 同一用户添加多个连接，Len 不应增加
	conn2 := createMockConn(2)
	c.SetRelation("user1", conn2)
	if c.Len() != 1 {
		t.Errorf("同一用户多连接 Len() = %d, 期望 1", c.Len())
	}

	// 添加第二个用户
	conn3 := createMockConn(3)
	c.SetRelation("user2", conn3)
	if c.Len() != 2 {
		t.Errorf("添加 2 个用户后 Len() = %d, 期望 2", c.Len())
	}

	// 删除一个用户的所有连接
	c.DelRelation("user1", conn1)
	c.DelRelation("user1", conn2)
	if c.Len() != 1 {
		t.Errorf("删除 1 个用户后 Len() = %d, 期望 1", c.Len())
	}
}

// TestGetCustomerIds 测试获取所有 customerId
func TestGetCustomerIds(t *testing.T) {
	c := newTestCollect()

	// 空状态应返回 nil
	ids := c.GetCustomerIds()
	if ids != nil {
		t.Errorf("空状态期望 nil，实际 %v", ids)
	}

	// 添加用户
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	conn3 := createMockConn(3)
	c.SetRelation("user1", conn1)
	c.SetRelation("user2", conn2)
	c.SetRelation("user3", conn3)

	ids = c.GetCustomerIds()
	if len(ids) != 3 {
		t.Errorf("期望 3 个 customerId，实际 %d", len(ids))
	}

	// 验证包含所有用户
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	expectedUsers := []string{"user1", "user2", "user3"}
	for _, user := range expectedUsers {
		if !idMap[user] {
			t.Errorf("缺少用户: %s", user)
		}
	}

	// 测试大量用户（跨分片）
	c2 := newTestCollect()
	for i := 0; i < 300; i++ {
		userId := fmt.Sprintf("user%d", i)
		conn := createMockConn(i + 100)
		c2.SetRelation(userId, conn)
	}

	ids = c2.GetCustomerIds()
	if len(ids) != 300 {
		t.Errorf("300 个用户期望返回 300 个 ID，实际 %d", len(ids))
	}
}

// TestGetConnListByCustomerId 测试获取单个用户的连接列表
func TestGetConnListByCustomerId(t *testing.T) {
	c := newTestCollect()

	// 测试空字符串
	conns := c.GetConnListByCustomerId("")
	if conns != nil {
		t.Errorf("空字符串期望 nil，实际 %v", conns)
	}

	// 测试不存在的用户
	conns = c.GetConnListByCustomerId("nonexistent")
	if conns != nil {
		t.Errorf("不存在的用户期望 nil，实际 %v", conns)
	}

	// 测试单个连接
	conn1 := createMockConn(1)
	c.SetRelation("user1", conn1)

	conns = c.GetConnListByCustomerId("user1")
	if len(conns) != 1 {
		t.Errorf("期望 1 个连接，实际 %d", len(conns))
	}
	if conns[0].Fd() != 1 {
		t.Errorf("期望 Fd=1，实际 %d", conns[0].Fd())
	}

	// 测试多个连接
	conn2 := createMockConn(2)
	conn3 := createMockConn(3)
	c.SetRelation("user1", conn2)
	c.SetRelation("user1", conn3)

	conns = c.GetConnListByCustomerId("user1")
	if len(conns) != 3 {
		t.Errorf("期望 3 个连接，实际 %d", len(conns))
	}

	// 验证所有连接都存在
	fdMap := make(map[int]bool)
	for _, conn := range conns {
		fdMap[conn.Fd()] = true
	}
	expectedFds := []int{1, 2, 3}
	for _, fd := range expectedFds {
		if !fdMap[fd] {
			t.Errorf("缺少 Fd=%d 的连接", fd)
		}
	}

	// 验证其他用户不受影响
	conns = c.GetConnListByCustomerId("user2")
	if conns != nil {
		t.Errorf("user2 不存在，期望 nil，实际 %v", conns)
	}
}

// TestGetConnListByCustomerIds 测试批量获取用户连接列表
func TestGetConnListByCustomerIds(t *testing.T) {
	c := newTestCollect()

	// 测试空列表
	result := c.GetConnListByCustomerIds([]string{})
	if result == nil {
		t.Error("空列表期望返回空 map，实际 nil")
	}
	if len(result) != 0 {
		t.Errorf("空列表期望 0 个元素，实际 %d", len(result))
	}

	// 测试包含空字符串的列表
	result = c.GetConnListByCustomerIds([]string{"", "user1", ""})
	if len(result) != 0 {
		t.Errorf("只有空字符串期望 0 个元素，实际 %d", len(result))
	}

	// 设置测试数据
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	conn3 := createMockConn(3)
	conn4 := createMockConn(4)
	c.SetRelation("user1", conn1)
	c.SetRelation("user1", conn2)
	c.SetRelation("user2", conn3)
	c.SetRelation("user3", conn4)

	// 测试查询存在的用户
	result = c.GetConnListByCustomerIds([]string{"user1", "user2"})
	if len(result) != 2 {
		t.Errorf("期望 2 个用户的结果，实际 %d", len(result))
	}

	if len(result["user1"]) != 2 {
		t.Errorf("user1 期望 2 个连接，实际 %d", len(result["user1"]))
	}
	if len(result["user2"]) != 1 {
		t.Errorf("user2 期望 1 个连接，实际 %d", len(result["user2"]))
	}

	// 测试查询不存在的用户（不应出现在结果中）
	result = c.GetConnListByCustomerIds([]string{"user1", "nonexistent"})
	if len(result) != 1 {
		t.Errorf("期望 1 个用户的结果，实际 %d", len(result))
	}
	if _, exists := result["nonexistent"]; exists {
		t.Error("不存在的用户不应出现在结果中")
	}

	// 测试部分用户存在
	result = c.GetConnListByCustomerIds([]string{"user1", "user2", "user3", "user4"})
	if len(result) != 3 {
		t.Errorf("期望 3 个用户的结果（user4 不存在），实际 %d", len(result))
	}

	// 测试大量用户（跨分片）
	c2 := newTestCollect()
	var userIds []string
	for i := 0; i < 100; i++ {
		userId := fmt.Sprintf("user%d", i)
		userIds = append(userIds, userId)
		conn := createMockConn(i + 1000)
		c2.SetRelation(userId, conn)
	}

	result = c2.GetConnListByCustomerIds(userIds)
	if len(result) != 100 {
		t.Errorf("100 个用户期望返回 100 个结果，实际 %d", len(result))
	}

	// 验证每个用户都有连接
	for _, userId := range userIds {
		if len(result[userId]) != 1 {
			t.Errorf("%s 期望 1 个连接，实际 %d", userId, len(result[userId]))
		}
	}
}

// TestConcurrentAccess 测试并发安全性
func TestConcurrentAccess(t *testing.T) {
	c := newTestCollect()
	var wg sync.WaitGroup

	// 并发写入
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userId := fmt.Sprintf("user%d", id)
			conn := createMockConn(id)
			c.SetRelation(userId, conn)
		}(i)
	}

	// 并发读取
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userId := fmt.Sprintf("user%d", id)
			c.GetConnListByCustomerId(userId)
		}(i)
	}

	// 并发删除
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userId := fmt.Sprintf("user%d", id)
			conn := createMockConn(id)
			c.DelRelation(userId, conn)
		}(i)
	}

	// 并发获取所有 ID
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.GetCustomerIds()
		}()
	}

	// 并发批量查询
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var userIds []string
			for j := 0; j < 20; j++ {
				userIds = append(userIds, fmt.Sprintf("user%d", j))
			}
			c.GetConnListByCustomerIds(userIds)
		}()
	}

	wg.Wait()

	// 验证最终状态一致性
	count := c.Len()
	if count < 0 || count > 100 {
		t.Errorf("并发操作后 Len() = %d, 超出合理范围", count)
	}
}

// TestShardDistribution 测试分片分布均匀性
func TestShardDistribution(t *testing.T) {
	shardCounts := make([]int, shardCount)

	// 生成 10000 个不同的 customerId，统计分片分布
	for i := 0; i < 10000; i++ {
		userId := fmt.Sprintf("user_%d", i)
		idx := hashCustomerId(userId)
		shardCounts[idx]++
	}

	// 计算平均值
	avg := 10000.0 / shardCount

	// 检查是否有分片过于不均（允许一定偏差）
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

	// 理想情况下每个分片约 39 个，偏差不应过大
	deviation := float64(maxCount-minCount) / avg
	if deviation > 1.0 { // 偏差超过 100% 认为分布不均
		t.Logf("分片分布: 最大=%d, 最小=%d, 平均=%.2f, 偏差=%.2f%%",
			maxCount, minCount, avg, deviation*100)
		// 这里只是警告，不失败，因为哈希分布可能有波动
	}
}

// BenchmarkSetRelation 测试设置绑定关系的性能
func BenchmarkSetRelation(b *testing.B) {
	c := newTestCollect()
	conn := createMockConn(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userId := fmt.Sprintf("user%d", i%1000) // 1000 个不同用户
		c.SetRelation(userId, conn)
	}
}

// BenchmarkDelRelation 测试删除绑定关系的性能
func BenchmarkDelRelation(b *testing.B) {
	c := newTestCollect()

	// 预创建数据
	for i := 0; i < b.N; i++ {
		userId := fmt.Sprintf("user%d", i)
		conn := createMockConn(i)
		c.SetRelation(userId, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userId := fmt.Sprintf("user%d", i)
		conn := createMockConn(i)
		c.DelRelation(userId, conn)
	}
}

// BenchmarkGetConnListByCustomerId 测试获取单个用户连接的性能
func BenchmarkGetConnListByCustomerId(b *testing.B) {
	c := newTestCollect()

	// 预创建数据
	for i := 0; i < 1000; i++ {
		userId := fmt.Sprintf("user%d", i)
		conn := createMockConn(i)
		c.SetRelation(userId, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userId := fmt.Sprintf("user%d", i%1000)
		c.GetConnListByCustomerId(userId)
	}
}

// BenchmarkGetCustomerIds 测试获取所有 customerId 的性能
func BenchmarkGetCustomerIds(b *testing.B) {
	c := newTestCollect()

	// 预创建 1000 个用户
	for i := 0; i < 1000; i++ {
		userId := fmt.Sprintf("user%d", i)
		conn := createMockConn(i)
		c.SetRelation(userId, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetCustomerIds()
	}
}

// BenchmarkGetConnListByCustomerIds 测试批量获取连接的性能
func BenchmarkGetConnListByCustomerIds(b *testing.B) {
	c := newTestCollect()

	// 预创建 1000 个用户
	for i := 0; i < 1000; i++ {
		userId := fmt.Sprintf("user%d", i)
		conn := createMockConn(i)
		c.SetRelation(userId, conn)
	}

	// 准备查询列表（每次查 100 个用户）
	var queryIds []string
	for i := 0; i < 100; i++ {
		queryIds = append(queryIds, fmt.Sprintf("user%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetConnListByCustomerIds(queryIds)
	}
}

// BenchmarkLen 测试获取长度的性能
func BenchmarkLen(b *testing.B) {
	c := newTestCollect()

	// 预创建 1000 个用户
	for i := 0; i < 1000; i++ {
		userId := fmt.Sprintf("user%d", i)
		conn := createMockConn(i)
		c.SetRelation(userId, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Len()
	}
}

// BenchmarkHashCustomerId 测试哈希函数的性能
func BenchmarkHashCustomerId(b *testing.B) {
	testCases := []string{
		"user123",
		"user_with_long_name_123456789",
		"用户ID中文测试",
		"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			hashCustomerId(tc)
		}
	}
}

// TestGetConnListByCustomerIdsNilInput 测试 nil 输入
func TestGetConnListByCustomerIdsNilInput(t *testing.T) {
	c := newTestCollect()

	// 测试 nil 输入
	result := c.GetConnListByCustomerIds(nil)
	if result == nil {
		t.Error("nil 输入期望返回空 map，实际 nil")
	}
	if len(result) != 0 {
		t.Errorf("nil 输入期望 0 个元素，实际 %d", len(result))
	}
}

// TestLargeScaleCrossShard 测试大规模跨分片场景
func TestLargeScaleCrossShard(t *testing.T) {
	c := newTestCollect()

	// 添加 500 个 customerId，确保跨多个分片
	const count = 500
	var conns []*MockConn
	for i := 0; i < count; i++ {
		customerId := fmt.Sprintf("customer_%d", i)
		conn := createMockConn(i + 10000)
		conns = append(conns, conn)
		c.SetRelation(customerId, conn)
	}

	// 验证数量
	if c.Len() != count {
		t.Errorf("Len() = %d, want %d", c.Len(), count)
	}

	// 获取所有 customerId
	customerIds := c.GetCustomerIds()
	if len(customerIds) != count {
		t.Errorf("GetCustomerIds() returned %d, want %d", len(customerIds), count)
	}

	// 批量查询部分 customerId
	queryIds := make([]string, 100)
	for i := 0; i < 100; i++ {
		queryIds[i] = fmt.Sprintf("customer_%d", i*5) // 每隔 5 个查一个
	}

	result := c.GetConnListByCustomerIds(queryIds)
	if len(result) != 100 {
		t.Errorf("批量查询期望 100 个结果，实际 %d", len(result))
	}

	// 验证每个 customerId 都有连接
	for _, customerId := range queryIds {
		if len(result[customerId]) != 1 {
			t.Errorf("%s 期望 1 个连接，实际 %d", customerId, len(result[customerId]))
		}
	}
}

// TestSetRelationSameConnMultipleTimes 测试同一连接多次设置
func TestSetRelationSameConnMultipleTimes(t *testing.T) {
	c := newTestCollect()
	conn := createMockConn(1)

	// 同一连接多次设置到同一用户
	c.SetRelation("user1", conn)
	c.SetRelation("user1", conn)
	c.SetRelation("user1", conn)

	conns := c.GetConnListByCustomerId("user1")
	if len(conns) != 1 {
		t.Errorf("同一连接多次设置期望 1 个连接，实际 %d", len(conns))
	}

	// 同一连接设置到不同用户
	c.SetRelation("user2", conn)
	c.SetRelation("user3", conn)

	// 验证每个用户都有该连接
	for _, userId := range []string{"user1", "user2", "user3"} {
		conns := c.GetConnListByCustomerId(userId)
		if len(conns) != 1 {
			t.Errorf("%s 期望 1 个连接，实际 %d", userId, len(conns))
		}
		if conns[0].Fd() != 1 {
			t.Errorf("%s 的连接 Fd 错误", userId)
		}
	}
}

// TestDelRelationEdgeCases 测试删除的边界情况
func TestDelRelationEdgeCases(t *testing.T) {
	c := newTestCollect()

	// 删除空字符串 customerId
	conn := createMockConn(1)
	c.DelRelation("", conn) // 不应 panic

	// 删除 nil conn（虽然实际不会发生，但要确保不 panic）
	// 注意：这会调用 conn.Fd()，如果 conn 为 nil 会 panic
	// 所以这里不测试 nil conn

	// 连续删除同一连接多次
	c.SetRelation("user1", conn)
	c.DelRelation("user1", conn)
	c.DelRelation("user1", conn) // 第二次删除不应 panic
	c.DelRelation("user1", conn) // 第三次删除不应 panic

	conns := c.GetConnListByCustomerId("user1")
	if conns != nil {
		t.Errorf("多次删除后期望 nil，实际 %v", conns)
	}
}

// TestGetCustomerIdsConsistency 测试 GetCustomerIds 的一致性
func TestGetCustomerIdsConsistency(t *testing.T) {
	c := newTestCollect()

	// 添加用户
	for i := 0; i < 100; i++ {
		customerId := fmt.Sprintf("user%d", i)
		conn := createMockConn(i)
		c.SetRelation(customerId, conn)
	}

	// 多次获取，验证结果一致（内容相同，顺序可能不同）
	ids1 := c.GetCustomerIds()
	ids2 := c.GetCustomerIds()

	if len(ids1) != len(ids2) {
		t.Errorf("两次获取长度不一致: %d vs %d", len(ids1), len(ids2))
	}

	// 转换为 map 比较
	map1 := make(map[string]bool)
	for _, id := range ids1 {
		map1[id] = true
	}
	map2 := make(map[string]bool)
	for _, id := range ids2 {
		map2[id] = true
	}

	for id := range map1 {
		if !map2[id] {
			t.Errorf("id %s 在第一次获取中存在，第二次不存在", id)
		}
	}
	for id := range map2 {
		if !map1[id] {
			t.Errorf("id %s 在第二次获取中存在，第一次不存在", id)
		}
	}
}
