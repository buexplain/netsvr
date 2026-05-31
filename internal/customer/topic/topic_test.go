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

// TestTopic_BasicOperations 测试基本操作
func TestTopic_BasicOperations(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 初始状态
	if topic.Len() != 0 {
		t.Errorf("初始长度期望=0, 实际=%d", topic.Len())
	}

	topics := topic.Get()
	if topics == nil {
		t.Error("空状态期望返回空切片而非nil")
	}

	counts := topic.CountConn()
	if len(counts) != 0 {
		t.Errorf("空状态期望主题数=0, 实际=%d", len(counts))
	}
}

// TestTopic_SetRelation 测试设置关系
func TestTopic_SetRelation(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 手动创建测试数据
	topicName := "topic1"
	conn := createTestConn(1)
	topic.SetRelation([]string{topicName}, conn)

	// 验证
	if topic.Len() != 1 {
		t.Errorf("期望主题数=1, 实际=%d", topic.Len())
	}

	sp := slicePool.NewWsConn(16)
	connList := topic.GetConnListByTopic(topicName, sp)
	if connList == nil {
		t.Fatal("期望获取到连接列表, 实际为nil")
	}
	if len(*connList) != 1 {
		t.Errorf("期望连接数=1, 实际=%d", len(*connList))
	}
	sp.Put(connList)
}

// TestTopic_DelRelation 测试删除关系
func TestTopic_DelRelation(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	topicName := "topic1"
	// 添加两个连接
	conn1 := createTestConn(1)
	conn2 := createTestConn(2)
	topic.SetRelation([]string{topicName}, conn1)
	topic.SetRelation([]string{topicName}, conn2)

	if topic.Len() != 1 {
		t.Fatalf("添加后期望主题数=1, 实际=%d", topic.Len())
	}

	// 删除第一个连接
	topic.DelRelationBySlice([]string{topicName}, conn1)

	// 再删除第二个连接
	topic.DelRelationBySlice([]string{topicName}, conn2)

	if topic.Len() != 0 {
		t.Errorf("删除所有连接后期望主题数=0, 实际=%d", topic.Len())
	}
}

// TestTopic_Del 测试删除主题
func TestTopic_Del(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	topicName := "topic1"
	// 添加主题和连接
	conn1 := createTestConn(1)
	conn2 := createTestConn(2)
	topic.SetRelation([]string{topicName}, conn1)
	topic.SetRelation([]string{topicName}, conn2)

	if topic.Len() != 1 {
		t.Fatalf("添加后期望主题数=1, 实际=%d", topic.Len())
	}

	// 删除主题
	deletedTopics := topic.Del([]string{topicName})
	if deletedTopics == nil {
		t.Fatal("期望返回删除的主题, 实际为nil")
	}

	if len(deletedTopics) != 1 {
		t.Errorf("期望删除1个主题, 实际=%d", len(deletedTopics))
	}

	if len(deletedTopics[topicName]) != 2 {
		t.Errorf("topic1期望连接数=2, 实际=%d", len(deletedTopics[topicName]))
	}

	// 验证主题已被删除
	if topic.Len() != 0 {
		t.Errorf("删除后期望剩余主题数=0, 实际=%d", topic.Len())
	}
}

// TestTopic_CountConn 测试统计连接数
func TestTopic_CountConn(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	topicNames := []string{"topic1", "topic2"}
	// 添加主题和连接
	for _, name := range topicNames {
		conn := createTestConn(1)
		topic.SetRelation([]string{name}, conn)
		if name == "topic1" {
			conn2 := createTestConn(2)
			topic.SetRelation([]string{name}, conn2)
		}
	}

	counts := topic.CountConn()
	if len(counts) != 2 {
		t.Errorf("期望主题数=2, 实际=%d", len(counts))
	}

	if counts["topic1"] != 2 {
		t.Errorf("topic1期望连接数=2, 实际=%d", counts["topic1"])
	}
	if counts["topic2"] != 1 {
		t.Errorf("topic2期望连接数=1, 实际=%d", counts["topic2"])
	}
}

// TestTopic_CountConnByTopic 测试统计指定主题的连接数
func TestTopic_CountConnByTopic(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 添加主题
	topicName := "topic1"
	conn := createTestConn(1)
	topic.SetRelation([]string{topicName}, conn)
	conn2 := createTestConn(2)
	topic.SetRelation([]string{topicName}, conn2)

	// 测试空topics
	counts := topic.CountConnByTopic([]string{})
	if counts != nil {
		t.Errorf("空topics期望返回nil, 实际=%v", counts)
	}

	// 测试统计指定主题
	counts = topic.CountConnByTopic([]string{topicName})
	if len(counts) != 1 {
		t.Errorf("期望统计1个主题, 实际=%d", len(counts))
	}

	if counts[topicName] != 2 {
		t.Errorf("topic1期望连接数=2, 实际=%d", counts[topicName])
	}

	// 测试不存在的主题
	counts = topic.CountConnByTopic([]string{"nonexistent"})
	if len(counts) != 0 {
		t.Errorf("不存在的主题期望返回空map, 实际=%v", counts)
	}
}

// TestTopic_GetConnListByTopics 测试获取多个主题的连接列表
func TestTopic_GetConnListByTopics(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 添加主题
	topicNames := []string{"topic1", "topic2"}
	for _, name := range topicNames {
		conn := createTestConn(1)
		topic.SetRelation([]string{name}, conn)
		if name == "topic1" {
			conn2 := createTestConn(2)
			topic.SetRelation([]string{name}, conn2)
		}
	}

	// 测试空topics
	result := topic.GetConnListByTopics([]string{})
	if result != nil {
		t.Errorf("空topics期望返回nil, 实际=%v", result)
	}

	// 获取多个主题
	result = topic.GetConnListByTopics(topicNames)
	if len(result) != 2 {
		t.Errorf("期望主题数=2, 实际=%d", len(result))
	}

	if len(result["topic1"]) != 2 {
		t.Errorf("topic1期望连接数=2, 实际=%d", len(result["topic1"]))
	}
	if len(result["topic2"]) != 1 {
		t.Errorf("topic2期望连接数=1, 实际=%d", len(result["topic2"]))
	}
}

// TestTopic_ConcurrentAccess 测试并发安全性
func TestTopic_ConcurrentAccess(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	var wg sync.WaitGroup
	concurrentOps := 100

	// 并发写入
	for i := 0; i < concurrentOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn := createTestConn(uint64(id))
			topicName := string(rune('A' + id%26))
			topic.SetRelation([]string{topicName}, conn)
		}(i)
	}

	// 并发读取
	for i := 0; i < concurrentOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			topicName := string(rune('A' + id%26))
			topic.CountConnByTopic([]string{topicName})
		}(i)
	}

	wg.Wait()

	// 验证没有panic且数据正确
	totalTopics := topic.Len()
	if totalTopics == 0 {
		t.Error("并发写入后主题数不应为0")
	}
}

// TestTopic_HashDistribution 测试哈希分布
func TestTopic_HashDistribution(t *testing.T) {
	shardCounts := make([]int, shardCount)
	testTopics := []string{
		"topic1", "topic2", "topic3", "test_topic",
		"topic_123", "abc", "xyz", "hello_world",
		"foo", "bar", "baz", "qux",
	}

	for _, topicName := range testTopics {
		idx := hashTopic(topicName)
		if idx < 0 || idx >= shardCount {
			t.Errorf("hashTopic(%s) = %d, 超出范围 [0, %d)", topicName, idx, shardCount)
		}
		shardCounts[idx]++
	}

	t.Logf("哈希分布情况: %+v", shardCounts)
}

// TestTopic_DelRelationByMap 测试使用map删除关系
func TestTopic_DelRelationByMap(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 空topics应该直接返回
	conn := createTestConn(1)
	topic.DelRelationByMap(nil, conn)
	topic.DelRelationByMap(map[string]struct{}{}, conn)

	// 添加主题和连接
	topicName := "topic1"
	topic.SetRelation([]string{topicName}, conn)
	if topic.Len() != 1 {
		t.Fatalf("添加后期望主题数=1, 实际=%d", topic.Len())
	}

	// 使用map删除关系
	topics := map[string]struct{}{
		"topic1": {},
	}
	topic.DelRelationByMap(topics, conn)
	if topic.Len() != 0 {
		t.Errorf("删除后期望主题数=0, 实际=%d", topic.Len())
	}
}

// TestTopic_GetConnList 测试获取所有主题的连接列表
func TestTopic_GetConnList(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 空状态
	result := topic.GetConnList()
	if len(result) != 0 {
		t.Errorf("空状态期望返回空map, 实际长度=%d", len(result))
	}

	// 添加多个主题
	topicNames := []string{"topic1", "topic2"}
	for _, name := range topicNames {
		conn := createTestConn(1)
		topic.SetRelation([]string{name}, conn)
		if name == "topic1" {
			conn2 := createTestConn(2)
			topic.SetRelation([]string{name}, conn2)
		}
	}

	result = topic.GetConnList()
	if len(result) != 2 {
		t.Errorf("期望主题数=2, 实际=%d", len(result))
	}

	if len(result["topic1"]) != 2 {
		t.Errorf("topic1期望连接数=2, 实际=%d", len(result["topic1"]))
	}
	if len(result["topic2"]) != 1 {
		t.Errorf("topic2期望连接数=1, 实际=%d", len(result["topic2"]))
	}
}

// TestTopic_DelEmptyTopics 测试删除空topics
func TestTopic_DelEmptyTopics(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	// 删除空topics应该返回nil
	result := topic.Del([]string{})
	if result != nil {
		t.Errorf("删除空topics期望返回nil, 实际=%v", result)
	}

	// 删除包含空字符串的topics会返回空map(因为len(topics) > 0, 会创建map)
	result = topic.Del([]string{""})
	if result == nil {
		t.Error("删除包含空字符串的topics期望返回空map而非nil")
	}
	if len(result) != 0 {
		t.Errorf("删除包含空字符串的topics期望返回空map, 实际长度=%d", len(result))
	}
}

// TestTopic_DelRelationBySliceEmptyTopics 测试DelRelationBySlice的空topics
func TestTopic_DelRelationBySliceEmptyTopics(t *testing.T) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	conn := createTestConn(1)

	// 空topics应该直接返回
	topic.DelRelationBySlice([]string{}, conn)
	topic.DelRelationBySlice(nil, conn)

	// 包含空字符串的topics
	topic.SetRelation([]string{"topic1"}, conn)
	topic.DelRelationBySlice([]string{""}, conn)
	if topic.Len() != 1 {
		t.Errorf("删除空字符串后期望主题数仍为1, 实际=%d", topic.Len())
	}
}

// BenchmarkTopic_Operations 性能基准测试
func BenchmarkTopic_Operations(b *testing.B) {
	topic := &collect{}
	for i := range topic.shards {
		topic.shards[i].data = make(map[string]map[uint64]*wsServer.Conn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := createTestConn(uint64(i))
		topicName := string(rune('A' + i%26))
		topic.SetRelation([]string{topicName}, conn)
	}
}
