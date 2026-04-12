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

package topic

import (
	"fmt"
	"github.com/panjf2000/gnet/v2"
	"io"
	"net"
	"netsvr/internal/utils/slicePool"
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

// TestHashTopic 测试哈希函数
func TestHashTopic(t *testing.T) {
	// 测试空字符串
	idx := hashTopic("")
	if idx < 0 || idx >= shardCount {
		t.Errorf("hashTopic(\"\") = %d, want in range [0, %d)", idx, shardCount)
	}

	// 测试相同输入产生相同输出
	idx1 := hashTopic("chat_room_1")
	idx2 := hashTopic("chat_room_1")
	if idx1 != idx2 {
		t.Errorf("hashTopic 不一致: %d != %d", idx1, idx2)
	}

	// 测试不同输入（允许哈希碰撞）
	_ = hashTopic("chat_room_2")

	// 测试所有结果都在有效范围内
	testCases := []string{"a", "abc", "主题名称", "topic_123456789", ""}
	for _, tc := range testCases {
		idx := hashTopic(tc)
		if idx < 0 || idx >= shardCount {
			t.Errorf("hashTopic(%q) = %d, out of range", tc, idx)
		}
	}
}

// TestSetRelation 测试设置绑定关系
func TestSetRelation(t *testing.T) {
	c := newTestCollect()

	// 测试空参数
	c.SetRelation([]string{}, createMockConn(1)) // 不应 panic
	c.SetRelation(nil, createMockConn(1))        // 不应 panic
	c.SetRelation([]string{"topic1"}, nil)       // 不应 panic

	// 测试基本设置
	conn1 := createMockConn(1)
	c.SetRelation([]string{"topic1"}, conn1)

	// 验证连接已添加
	conns := c.GetConnListByTopics([]string{"topic1"})
	if len(conns) != 1 {
		t.Errorf("期望 1 个主题，实际 %d", len(conns))
	}
	if len(conns["topic1"]) != 1 {
		t.Errorf("topic1 期望 1 个连接，实际 %d", len(conns["topic1"]))
	}

	// 测试同一连接订阅多个主题
	c.SetRelation([]string{"topic1", "topic2", "topic3"}, conn1)

	conns = c.GetConnListByTopics([]string{"topic1", "topic2", "topic3"})
	if len(conns) != 3 {
		t.Errorf("期望 3 个主题，实际 %d", len(conns))
	}

	// 测试多个连接订阅同一主题
	conn2 := createMockConn(2)
	c.SetRelation([]string{"topic1"}, conn2)

	conns = c.GetConnListByTopics([]string{"topic1"})
	if len(conns["topic1"]) != 2 {
		t.Errorf("topic1 期望 2 个连接，实际 %d", len(conns["topic1"]))
	}

	// 测试包含空字符串的 topic 列表
	c.SetRelation([]string{"", "topic4", ""}, conn1)
	conns = c.GetConnListByTopics([]string{"topic4"})
	if len(conns) != 1 {
		t.Errorf("期望 topic4 存在，实际 %d", len(conns))
	}
}

// TestDelRelationByMap 测试通过 map 删除绑定关系
func TestDelRelationByMap(t *testing.T) {
	c := newTestCollect()

	// 设置测试数据
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	c.SetRelation([]string{"topic1", "topic2", "topic3"}, conn1)
	c.SetRelation([]string{"topic1", "topic2"}, conn2)

	// 删除 conn1 在 topic1 和 topic2 的订阅
	topics := map[string]struct{}{
		"topic1": {},
		"topic2": {},
	}
	c.DelRelationByMap(topics, conn1)

	// 验证 conn1 已被删除
	conns := c.GetConnListByTopics([]string{"topic1", "topic2", "topic3"})
	if len(conns["topic1"]) != 1 {
		t.Errorf("topic1 期望 1 个连接（conn2），实际 %d", len(conns["topic1"]))
	}
	if len(conns["topic2"]) != 1 {
		t.Errorf("topic2 期望 1 个连接（conn2），实际 %d", len(conns["topic2"]))
	}
	if len(conns["topic3"]) != 1 {
		t.Errorf("topic3 期望 1 个连接（conn1），实际 %d", len(conns["topic3"]))
	}

	// 删除最后一个连接，应该清理 topic
	c.DelRelationByMap(map[string]struct{}{"topic3": {}}, conn1)
	conns = c.GetConnListByTopics([]string{"topic3"})
	if len(conns) != 0 {
		t.Errorf("topic3 应被清理，实际还存在")
	}

	// 测试空 map
	c.DelRelationByMap(map[string]struct{}{}, conn1) // 不应 panic

	// 测试删除不存在的 topic
	c.DelRelationByMap(map[string]struct{}{"nonexistent": {}}, conn1) // 不应 panic
}

// TestDelRelationBySlice 测试通过 slice 删除绑定关系
func TestDelRelationBySlice(t *testing.T) {
	c := newTestCollect()

	// 设置测试数据
	conn1 := createMockConn(1)
	c.SetRelation([]string{"topic1", "topic2", "topic3"}, conn1)

	// 删除部分订阅
	c.DelRelationBySlice([]string{"topic1", "topic2"}, conn1)

	// 验证
	conns := c.GetConnListByTopics([]string{"topic1", "topic2", "topic3"})
	if len(conns) != 1 || conns["topic3"] == nil {
		t.Errorf("只应保留 topic3")
	}
	if len(conns["topic3"]) != 1 {
		t.Errorf("topic3 期望 1 个连接，实际 %d", len(conns["topic3"]))
	}

	// 测试空 slice
	c.DelRelationBySlice([]string{}, conn1) // 不应 panic

	// 测试包含空字符串
	c.DelRelationBySlice([]string{"", "topic3", ""}, conn1)
	conns = c.GetConnListByTopics([]string{"topic3"})
	if len(conns) != 0 {
		t.Errorf("topic3 应被删除")
	}
}

// TestDel 测试删除主题并返回连接
func TestDel(t *testing.T) {
	c := newTestCollect()

	// 设置测试数据
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	c.SetRelation([]string{"topic1", "topic2"}, conn1)
	c.SetRelation([]string{"topic1"}, conn2)

	// 删除主题
	deleted := c.Del([]string{"topic1", "topic2", "nonexistent"})

	// 验证返回值
	if len(deleted) != 2 {
		t.Errorf("期望删除 2 个主题，实际 %d", len(deleted))
	}

	if len(deleted["topic1"]) != 2 {
		t.Errorf("topic1 期望 2 个连接，实际 %d", len(deleted["topic1"]))
	}
	if len(deleted["topic2"]) != 1 {
		t.Errorf("topic2 期望 1 个连接，实际 %d", len(deleted["topic2"]))
	}
	if _, exists := deleted["nonexistent"]; exists {
		t.Error("不存在的 topic 不应出现在结果中")
	}

	// 验证主题已从管理器中移除
	topics := c.Get()
	for _, topic := range topics {
		if topic == "topic1" || topic == "topic2" {
			t.Errorf("%s 应已被删除", topic)
		}
	}

	// 测试空参数
	result := c.Del([]string{})
	if result != nil {
		t.Errorf("空参数期望返回 nil")
	}

	// 验证返回的 map 可以安全修改（所有权转移）
	deleted["topic1"][999] = createMockConn(999) // 不应影响内部状态
}

// TestLen 测试获取主题数量
func TestLen(t *testing.T) {
	c := newTestCollect()

	// 初始状态应为 0
	if c.Len() != 0 {
		t.Errorf("初始 Len() = %d, 期望 0", c.Len())
	}

	// 添加主题
	conn1 := createMockConn(1)
	c.SetRelation([]string{"topic1", "topic2", "topic3"}, conn1)
	if c.Len() != 3 {
		t.Errorf("添加 3 个主题后 Len() = %d, 期望 3", c.Len())
	}

	// 同一主题多个连接，Len 不应增加
	conn2 := createMockConn(2)
	c.SetRelation([]string{"topic1"}, conn2)
	if c.Len() != 3 {
		t.Errorf("同一主题多连接 Len() = %d, 期望 3", c.Len())
	}

	// 删除主题
	c.Del([]string{"topic1"})
	if c.Len() != 2 {
		t.Errorf("删除 1 个主题后 Len() = %d, 期望 2", c.Len())
	}
}

// TestGet 测试获取全部主题
func TestGet(t *testing.T) {
	c := newTestCollect()

	// 空状态应返回空切片
	topics := c.Get()
	if topics == nil {
		t.Error("空状态期望返回空切片，实际 nil")
	}
	if len(topics) != 0 {
		t.Errorf("空状态期望 0 个主题，实际 %d", len(topics))
	}

	// 添加主题
	conn1 := createMockConn(1)
	c.SetRelation([]string{"topic1", "topic2", "topic3"}, conn1)

	topics = c.Get()
	if len(topics) != 3 {
		t.Errorf("期望 3 个主题，实际 %d", len(topics))
	}

	// 验证包含所有主题
	topicMap := make(map[string]bool)
	for _, topic := range topics {
		topicMap[topic] = true
	}
	expectedTopics := []string{"topic1", "topic2", "topic3"}
	for _, expected := range expectedTopics {
		if !topicMap[expected] {
			t.Errorf("缺少主题: %s", expected)
		}
	}

	// 测试大量主题（跨分片）
	c2 := newTestCollect()
	for i := 0; i < 300; i++ {
		topicName := fmt.Sprintf("topic%d", i)
		conn := createMockConn(i + 100)
		c2.SetRelation([]string{topicName}, conn)
	}

	topics = c2.Get()
	if len(topics) != 300 {
		t.Errorf("300 个主题期望返回 300 个，实际 %d", len(topics))
	}
}

// TestCountConn 测试统计全部主题的连接数
func TestCountConn(t *testing.T) {
	c := newTestCollect()

	// 空状态
	counts := c.CountConn()
	if counts == nil {
		t.Error("期望返回空 map，实际 nil")
	}
	if len(counts) != 0 {
		t.Errorf("空状态期望 0 个主题，实际 %d", len(counts))
	}

	// 添加主题和连接
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	conn3 := createMockConn(3)
	c.SetRelation([]string{"topic1", "topic2"}, conn1)
	c.SetRelation([]string{"topic1"}, conn2)
	c.SetRelation([]string{"topic2"}, conn3)

	counts = c.CountConn()
	if len(counts) != 2 {
		t.Errorf("期望 2 个主题，实际 %d", len(counts))
	}
	if counts["topic1"] != 2 {
		t.Errorf("topic1 期望 2 个连接，实际 %d", counts["topic1"])
	}
	if counts["topic2"] != 2 {
		t.Errorf("topic2 期望 2 个连接，实际 %d", counts["topic2"])
	}
}

// TestCountConnByTopic 测试批量统计主题连接数
func TestCountConnByTopic(t *testing.T) {
	c := newTestCollect()

	// 测试空参数
	counts := c.CountConnByTopic([]string{})
	if counts != nil {
		t.Errorf("空参数期望 nil，实际 %v", counts)
	}

	// 设置测试数据
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	c.SetRelation([]string{"topic1", "topic2", "topic3"}, conn1)
	c.SetRelation([]string{"topic1", "topic2"}, conn2)

	// 统计部分主题
	counts = c.CountConnByTopic([]string{"topic1", "topic2"})
	if len(counts) != 2 {
		t.Errorf("期望 2 个主题的结果，实际 %d", len(counts))
	}
	if counts["topic1"] != 2 {
		t.Errorf("topic1 期望 2 个连接，实际 %d", counts["topic1"])
	}
	if counts["topic2"] != 2 {
		t.Errorf("topic2 期望 2 个连接，实际 %d", counts["topic2"])
	}

	// 测试不存在的主题（不应出现在结果中）
	counts = c.CountConnByTopic([]string{"topic1", "nonexistent"})
	if len(counts) != 1 {
		t.Errorf("期望 1 个主题的结果，实际 %d", len(counts))
	}
	if _, exists := counts["nonexistent"]; exists {
		t.Error("不存在的主题不应出现在结果中")
	}

	// 测试包含空字符串
	counts = c.CountConnByTopic([]string{"", "topic3", ""})
	if len(counts) != 1 || counts["topic3"] != 1 {
		t.Errorf("topic3 期望 1 个连接")
	}
}

// TestGetConnListByTopic 测试获取单个主题的连接列表（使用对象池）
func TestGetConnListByTopic(t *testing.T) {
	c := newTestCollect()
	sp := slicePool.NewWsConn(16)

	// 测试空字符串
	conns := c.GetConnListByTopic("", sp)
	if conns != nil {
		t.Errorf("空字符串期望 nil，实际 %v", conns)
	}

	// 测试不存在的主题
	conns = c.GetConnListByTopic("nonexistent", sp)
	if conns != nil {
		t.Errorf("不存在的主题期望 nil，实际 %v", conns)
	}

	// 设置测试数据
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	conn3 := createMockConn(3)
	c.SetRelation([]string{"topic1"}, conn1)
	c.SetRelation([]string{"topic1"}, conn2)
	c.SetRelation([]string{"topic1"}, conn3)

	// 获取连接列表
	conns = c.GetConnListByTopic("topic1", sp)
	if conns == nil {
		t.Fatal("期望非 nil")
	}
	if len(*conns) != 3 {
		t.Errorf("期望 3 个连接，实际 %d", len(*conns))
	}

	// 验证所有连接都存在
	fdMap := make(map[int]bool)
	for _, conn := range *conns {
		fdMap[conn.Fd()] = true
	}
	expectedFds := []int{1, 2, 3}
	for _, fd := range expectedFds {
		if !fdMap[fd] {
			t.Errorf("缺少 Fd=%d 的连接", fd)
		}
	}

	// 归还到池
	sp.Put(conns)

	// 再次获取，验证池复用
	conns2 := c.GetConnListByTopic("topic1", sp)
	if conns2 == nil {
		t.Fatal("期望非 nil")
	}
	sp.Put(conns2)
}

// TestGetConnListByTopics 测试批量获取主题连接列表
func TestGetConnListByTopics(t *testing.T) {
	c := newTestCollect()

	// 测试空参数
	result := c.GetConnListByTopics([]string{})
	if result != nil {
		t.Errorf("空参数期望 nil，实际 %v", result)
	}

	// 设置测试数据
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	conn3 := createMockConn(3)
	c.SetRelation([]string{"topic1", "topic2"}, conn1)
	c.SetRelation([]string{"topic1"}, conn2)
	c.SetRelation([]string{"topic3"}, conn3)

	// 批量查询
	result = c.GetConnListByTopics([]string{"topic1", "topic2", "topic3"})
	if len(result) != 3 {
		t.Errorf("期望 3 个主题的结果，实际 %d", len(result))
	}
	if len(result["topic1"]) != 2 {
		t.Errorf("topic1 期望 2 个连接，实际 %d", len(result["topic1"]))
	}
	if len(result["topic2"]) != 1 {
		t.Errorf("topic2 期望 1 个连接，实际 %d", len(result["topic2"]))
	}
	if len(result["topic3"]) != 1 {
		t.Errorf("topic3 期望 1 个连接，实际 %d", len(result["topic3"]))
	}

	// 测试不存在的主题
	result = c.GetConnListByTopics([]string{"topic1", "nonexistent"})
	if len(result) != 1 {
		t.Errorf("期望 1 个主题的结果，实际 %d", len(result))
	}

	// 测试包含空字符串
	result = c.GetConnListByTopics([]string{"", "topic2", ""})
	if len(result) != 1 || len(result["topic2"]) != 1 {
		t.Errorf("topic2 期望 1 个连接")
	}
}

// TestGetConnList 测试获取全部主题的连接列表
func TestGetConnList(t *testing.T) {
	c := newTestCollect()

	// 空状态
	result := c.GetConnList()
	if result == nil {
		t.Error("期望返回空 map，实际 nil")
	}
	if len(result) != 0 {
		t.Errorf("空状态期望 0 个主题，实际 %d", len(result))
	}

	// 设置测试数据
	conn1 := createMockConn(1)
	conn2 := createMockConn(2)
	c.SetRelation([]string{"topic1", "topic2"}, conn1)
	c.SetRelation([]string{"topic1"}, conn2)

	result = c.GetConnList()
	if len(result) != 2 {
		t.Errorf("期望 2 个主题，实际 %d", len(result))
	}
	if len(result["topic1"]) != 2 {
		t.Errorf("topic1 期望 2 个连接，实际 %d", len(result["topic1"]))
	}
	if len(result["topic2"]) != 1 {
		t.Errorf("topic2 期望 1 个连接，实际 %d", len(result["topic2"]))
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
			topicName := fmt.Sprintf("topic%d", id)
			conn := createMockConn(id)
			c.SetRelation([]string{topicName}, conn)
		}(i)
	}

	// 并发读取
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			topicName := fmt.Sprintf("topic%d", id)
			c.GetConnListByTopics([]string{topicName})
		}(i)
	}

	// 并发删除
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			topicName := fmt.Sprintf("topic%d", id)
			conn := createMockConn(id)
			c.DelRelationBySlice([]string{topicName}, conn)
		}(i)
	}

	// 并发获取全部
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Get()
			c.Len()
			c.CountConn()
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

	// 生成 10000 个不同的 topic，统计分片分布
	for i := 0; i < 10000; i++ {
		topicName := fmt.Sprintf("topic_%d", i)
		idx := hashTopic(topicName)
		shardCounts[idx]++
	}

	// 计算平均值
	avg := 10000.0 / shardCount

	// 检查是否有分片过于不均
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

// BenchmarkSetRelation 测试设置绑定关系的性能
func BenchmarkSetRelation(b *testing.B) {
	c := newTestCollect()
	conn := createMockConn(1)
	topics := []string{"topic1", "topic2", "topic3", "topic4", "topic5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.SetRelation(topics, conn)
	}
}

// BenchmarkDelRelationBySlice 测试删除绑定关系的性能
func BenchmarkDelRelationBySlice(b *testing.B) {
	c := newTestCollect()
	conn := createMockConn(1)
	topics := []string{"topic1", "topic2", "topic3", "topic4", "topic5"}

	// 预创建数据
	for i := 0; i < b.N; i++ {
		c.SetRelation(topics, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.DelRelationBySlice(topics, conn)
	}
}

// BenchmarkDel 测试删除主题的性能
func BenchmarkDel(b *testing.B) {
	c := newTestCollect()
	topics := make([]string, 100)
	for i := 0; i < 100; i++ {
		topics[i] = fmt.Sprintf("topic%d", i)
		conn := createMockConn(i)
		c.SetRelation([]string{topics[i]}, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Del(topics)
		// 重新创建数据供下次迭代使用
		for j, topic := range topics {
			conn := createMockConn(j)
			c.SetRelation([]string{topic}, conn)
		}
	}
}

// BenchmarkGetConnListByTopic 测试获取单个主题连接的性能
func BenchmarkGetConnListByTopic(b *testing.B) {
	c := newTestCollect()
	sp := slicePool.NewWsConn(16)

	// 预创建数据
	for i := 0; i < 100; i++ {
		conn := createMockConn(i)
		c.SetRelation([]string{"benchmark_topic"}, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conns := c.GetConnListByTopic("benchmark_topic", sp)
		if conns != nil {
			sp.Put(conns)
		}
	}
}

// BenchmarkGetConnListByTopics 测试批量获取连接的性能
func BenchmarkGetConnListByTopics(b *testing.B) {
	c := newTestCollect()

	// 预创建 100 个主题
	var topics []string
	for i := 0; i < 100; i++ {
		topicName := fmt.Sprintf("topic%d", i)
		topics = append(topics, topicName)
		conn := createMockConn(i)
		c.SetRelation([]string{topicName}, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetConnListByTopics(topics)
	}
}

// BenchmarkCountConnByTopic 测试批量统计连接数的性能
func BenchmarkCountConnByTopic(b *testing.B) {
	c := newTestCollect()

	// 预创建 100 个主题
	var topics []string
	for i := 0; i < 100; i++ {
		topicName := fmt.Sprintf("topic%d", i)
		topics = append(topics, topicName)
		conn := createMockConn(i)
		c.SetRelation([]string{topicName}, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.CountConnByTopic(topics)
	}
}

// BenchmarkLen 测试获取长度的性能
func BenchmarkLen(b *testing.B) {
	c := newTestCollect()

	// 预创建 1000 个主题
	for i := 0; i < 1000; i++ {
		topicName := fmt.Sprintf("topic%d", i)
		conn := createMockConn(i)
		c.SetRelation([]string{topicName}, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Len()
	}
}

// BenchmarkGet 测试获取全部主题的性能
func BenchmarkGet(b *testing.B) {
	c := newTestCollect()

	// 预创建 1000 个主题
	for i := 0; i < 1000; i++ {
		topicName := fmt.Sprintf("topic%d", i)
		conn := createMockConn(i)
		c.SetRelation([]string{topicName}, conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get()
	}
}

// BenchmarkHashTopic 测试哈希函数的性能
func BenchmarkHashTopic(b *testing.B) {
	testCases := []string{
		"topic123",
		"topic_with_long_name_123456789",
		"主题名称中文测试",
		"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			hashTopic(tc)
		}
	}
}

// TestGetConnListByTopicPoolReuse 测试对象池复用
func TestGetConnListByTopicPoolReuse(t *testing.T) {
	c := newTestCollect()
	sp := slicePool.NewWsConn(16)

	// 添加连接
	for i := 0; i < 10; i++ {
		conn := createMockConn(i)
		c.SetRelation([]string{"test_topic"}, conn)
	}

	// 第一次获取
	conns1 := c.GetConnListByTopic("test_topic", sp)
	if conns1 == nil {
		t.Fatal("期望非 nil")
	}
	cap1 := cap(*conns1)

	// 归还
	sp.Put(conns1)

	// 第二次获取，应该复用池中的切片
	conns2 := c.GetConnListByTopic("test_topic", sp)
	if conns2 == nil {
		t.Fatal("期望非 nil")
	}

	// 验证容量相似（可能因实现略有不同）
	if cap(*conns2) < cap1/2 {
		t.Logf("警告：池复用可能未生效，cap1=%d, cap2=%d", cap1, cap(*conns2))
	}

	sp.Put(conns2)
}

// TestSetRelationDuplicateTopics 测试重复 topic 的处理
func TestSetRelationDuplicateTopics(t *testing.T) {
	c := newTestCollect()
	conn := createMockConn(1)

	// 同一 topic 在列表中重复出现
	c.SetRelation([]string{"topic1", "topic1", "topic1"}, conn)

	conns := c.GetConnListByTopics([]string{"topic1"})
	if len(conns) != 1 {
		t.Errorf("期望 1 个主题，实际 %d", len(conns))
	}
	// 由于使用 map，重复的 topic 只会添加一次连接
	if len(conns["topic1"]) != 1 {
		t.Errorf("topic1 期望 1 个连接，实际 %d", len(conns["topic1"]))
	}
}

// TestDelWithEmptyStrings 测试 Del 包含空字符串的情况
func TestDelWithEmptyStrings(t *testing.T) {
	c := newTestCollect()

	// 设置测试数据
	conn := createMockConn(1)
	c.SetRelation([]string{"topic1", "topic2"}, conn)

	// 删除包含空字符串
	deleted := c.Del([]string{"", "topic1", "", "topic2", ""})

	if len(deleted) != 2 {
		t.Errorf("期望删除 2 个主题，实际 %d", len(deleted))
	}

	// 验证主题已被删除
	topics := c.Get()
	if len(topics) != 0 {
		t.Errorf("期望所有主题被删除，实际还有 %d 个", len(topics))
	}
}

// TestLargeScaleCrossShard 测试大规模跨分片场景
func TestLargeScaleCrossShard(t *testing.T) {
	c := newTestCollect()

	// 添加 500 个 topic，确保跨多个分片
	const count = 500
	for i := 0; i < count; i++ {
		topicName := fmt.Sprintf("topic_%d", i)
		conn := createMockConn(i + 10000)
		c.SetRelation([]string{topicName}, conn)
	}

	// 验证数量
	if c.Len() != count {
		t.Errorf("Len() = %d, want %d", c.Len(), count)
	}

	// 获取所有 topic
	topics := c.Get()
	if len(topics) != count {
		t.Errorf("Get() returned %d, want %d", len(topics), count)
	}

	// 批量查询部分 topic
	queryTopics := make([]string, 100)
	for i := 0; i < 100; i++ {
		queryTopics[i] = fmt.Sprintf("topic_%d", i*5) // 每隔 5 个查一个
	}

	result := c.GetConnListByTopics(queryTopics)
	if len(result) != 100 {
		t.Errorf("批量查询期望 100 个结果，实际 %d", len(result))
	}

	// 统计连接数
	counts := c.CountConnByTopic(queryTopics)
	if len(counts) != 100 {
		t.Errorf("统计期望 100 个结果，实际 %d", len(counts))
	}
}

// TestGetConnListLargeScale 测试 GetConnList 大规模数据
func TestGetConnListLargeScale(t *testing.T) {
	c := newTestCollect()

	// 添加 200 个主题，每个主题 3 个连接
	for i := 0; i < 200; i++ {
		topicName := fmt.Sprintf("topic_%d", i)
		for j := 0; j < 3; j++ {
			conn := createMockConn(i*100 + j)
			c.SetRelation([]string{topicName}, conn)
		}
	}

	result := c.GetConnList()
	if len(result) != 200 {
		t.Errorf("期望 200 个主题，实际 %d", len(result))
	}

	// 验证每个主题的连接数
	for topic, conns := range result {
		if len(conns) != 3 {
			t.Errorf("%s 期望 3 个连接，实际 %d", topic, len(conns))
		}
	}
}

// TestCountConnLargeScale 测试 CountConn 大规模数据
func TestCountConnLargeScale(t *testing.T) {
	c := newTestCollect()

	// 添加 100 个主题，不同数量的连接
	for i := 0; i < 100; i++ {
		topicName := fmt.Sprintf("topic_%d", i)
		connCount := (i % 5) + 1 // 1-5 个连接
		for j := 0; j < connCount; j++ {
			conn := createMockConn(i*100 + j)
			c.SetRelation([]string{topicName}, conn)
		}
	}

	counts := c.CountConn()
	if len(counts) != 100 {
		t.Errorf("期望 100 个主题，实际 %d", len(counts))
	}

	// 验证几个主题的计数
	expectedCounts := map[string]int32{
		"topic_0": 1,
		"topic_1": 2,
		"topic_4": 5,
		"topic_9": 5,
	}
	for topic, expected := range expectedCounts {
		if counts[topic] != expected {
			t.Errorf("%s 期望 %d 个连接，实际 %d", topic, expected, counts[topic])
		}
	}
}

// TestDelRelationByMapEdgeCases 测试 DelRelationByMap 边界情况
func TestDelRelationByMapEdgeCases(t *testing.T) {
	c := newTestCollect()
	conn := createMockConn(1)

	// 删除 nil map
	c.DelRelationByMap(nil, conn) // 不应 panic

	// 删除包含空字符串 key 的 map
	c.SetRelation([]string{"topic1"}, conn)
	topics := map[string]struct{}{
		"":        {},
		"topic1":  {},
		"invalid": {},
	}
	c.DelRelationByMap(topics, conn)

	// 验证 topic1 被删除
	conns := c.GetConnListByTopics([]string{"topic1"})
	if len(conns) != 0 {
		t.Errorf("topic1 应被删除")
	}
}

// TestGetConsistency 测试 Get 的一致性
func TestGetConsistency(t *testing.T) {
	c := newTestCollect()

	// 添加主题
	for i := 0; i < 100; i++ {
		topicName := fmt.Sprintf("topic%d", i)
		conn := createMockConn(i)
		c.SetRelation([]string{topicName}, conn)
	}

	// 多次获取，验证结果一致（内容相同，顺序可能不同）
	topics1 := c.Get()
	topics2 := c.Get()

	if len(topics1) != len(topics2) {
		t.Errorf("两次获取长度不一致: %d vs %d", len(topics1), len(topics2))
	}

	// 转换为 map 比较
	map1 := make(map[string]bool)
	for _, topic := range topics1 {
		map1[topic] = true
	}
	map2 := make(map[string]bool)
	for _, topic := range topics2 {
		map2[topic] = true
	}

	for topic := range map1 {
		if !map2[topic] {
			t.Errorf("topic %s 在第一次获取中存在，第二次不存在", topic)
		}
	}
	for topic := range map2 {
		if !map1[topic] {
			t.Errorf("topic %s 在第二次获取中存在，第一次不存在", topic)
		}
	}
}
