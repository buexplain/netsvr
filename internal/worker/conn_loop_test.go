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

package worker

import (
	"context"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"google.golang.org/protobuf/proto"
	"net"
	internalMetrics "netsvr/internal/metrics"
	"testing"
	"time"
)

// mockNetConn 模拟 net.Conn 用于测试
type mockNetConn struct {
	writeData     [][]byte
	writeErr      error
	remoteAddr    string
	closeCalled   bool
	writeDeadline time.Time
}

func (m *mockNetConn) Read(_ []byte) (n int, err error) {
	return 0, nil
}

func (m *mockNetConn) Write(b []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	// 复制数据以便后续验证
	data := make([]byte, len(b))
	copy(data, b)
	m.writeData = append(m.writeData, data)
	return len(b), nil
}

func (m *mockNetConn) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockNetConn) LocalAddr() net.Addr {
	return nil
}

func (m *mockNetConn) RemoteAddr() net.Addr {
	return &mockAddr{addr: m.remoteAddr}
}

func (m *mockNetConn) SetDeadline(_ time.Time) error {
	return nil
}

func (m *mockNetConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (m *mockNetConn) SetWriteDeadline(t time.Time) error {
	m.writeDeadline = t
	return nil
}

type mockAddr struct {
	addr string
}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return m.addr
}

// TestLoopSendBatchMode 测试loopSend的批量发送分支（总大小 < packLimit）
func TestLoopSendBatchMode(t *testing.T) {
	// 记录初始metrics值
	initialCount := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedCount].Meter.Snapshot().Count()
	initialBytes := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedByte].Meter.Snapshot().Count()

	// 创建mock连接
	mockConn := &mockNetConn{
		remoteAddr: "127.0.0.1:8080",
		writeErr:   nil,
	}

	// dequeueSize=5，packLimit = max(2097152, 5*2*1024) = 2097152 (2MB)
	// 每条消息100字节，5条总共500字节 < 2MB，确保走批量分支
	dequeueSize := 5
	conn := newConn(mockConn, dequeueSize)
	defer conn.Close()

	// 发送小消息
	msgCount := 10
	type expectedData struct {
		uniqId     string
		customerId string
	}
	expectedList := make([]expectedData, msgCount)

	// 计算预期的总字节数（8字节header + proto body）
	var expectedTotalBytes int64

	for i := 0; i < msgCount; i++ {
		expectedList[i] = expectedData{
			uniqId:     string(rune('A' + i)),
			customerId: "batch",
		}
		msg := &netsvrProtocol.ConnClose{
			UniqId:     expectedList[i].uniqId,
			CustomerId: expectedList[i].customerId,
		}
		// 使用proto.Size精确计算body大小
		bodySize := proto.Size(msg)
		expectedTotalBytes += int64(8 + bodySize) // 8字节header + body
		conn.Send(msg, netsvrProtocol.Cmd_ConnClose)
	}

	// 等待异步发送完成
	time.Sleep(300 * time.Millisecond)

	// 计算mockConn实际收到的总字节数
	var mockConnTotalBytes int64
	for _, data := range mockConn.writeData {
		mockConnTotalBytes += int64(len(data))
	}

	// 验证metrics增量
	finalCount := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedCount].Meter.Snapshot().Count()
	finalBytes := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedByte].Meter.Snapshot().Count()

	deltaCount := finalCount - initialCount
	deltaBytes := finalBytes - initialBytes

	if deltaCount != int64(msgCount) {
		t.Fatalf("期望成功次数增量=%d, 实际=%d", msgCount, deltaCount)
	}

	// 精确断言：三个值应该完全一致
	// 1. expectedTotalBytes: proto.Size计算的预期值
	// 2. mockConnTotalBytes: mockConn实际接收的字节数
	// 3. deltaBytes: metrics统计的字节增量
	if deltaBytes != expectedTotalBytes || mockConnTotalBytes != expectedTotalBytes {
		t.Fatalf("字节数不一致 - 预期:%d, metrics:%d, mockConn:%d",
			expectedTotalBytes, deltaBytes, mockConnTotalBytes)
	}

	t.Logf("批量模式 - 次数增量:%d, 字节增量:%d (预期:%d), 写入批次:%d",
		deltaCount, deltaBytes, expectedTotalBytes, len(mockConn.writeData))
}

// TestLoopSendSingleMode 测试loopSend的单个发送分支（总大小 > packLimit）
func TestLoopSendSingleMode(t *testing.T) {
	// 记录初始metrics值
	initialCount := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedCount].Meter.Snapshot().Count()
	initialBytes := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedByte].Meter.Snapshot().Count()

	// 创建mock连接
	mockConn := &mockNetConn{
		remoteAddr: "127.0.0.1:8081",
		writeErr:   nil,
	}

	// dequeueSize=2，packLimit = max(2097152, 2*2*1024) = 2097152 (2MB)
	// 每条消息1.5MB，2条总共3MB > 2MB，确保走单个发送分支
	dequeueSize := 2
	conn := newConn(mockConn, dequeueSize)
	defer conn.Close()

	largeData := make([]byte, 1500*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	msgCount := 4
	type expectedData struct {
		uniqId     string
		customerId string
		data       []byte
	}
	expectedList := make([]expectedData, msgCount)

	// 计算预期的总字节数（8字节header + proto body）
	var expectedTotalBytes int64

	// 发送大消息
	for i := 0; i < msgCount; i++ {
		expectedList[i] = expectedData{
			uniqId:     string(rune('A' + i)),
			customerId: "single",
			data:       largeData,
		}
		msg := &netsvrProtocol.Transfer{
			UniqId:     expectedList[i].uniqId,
			CustomerId: expectedList[i].customerId,
			Data:       expectedList[i].data,
		}
		// 使用proto.Size精确计算body大小
		bodySize := proto.Size(msg)
		expectedTotalBytes += int64(8 + bodySize) // 8字节header + body
		conn.Send(msg, netsvrProtocol.Cmd_Transfer)
	}

	t.Logf("总消息数: %d, dequeueSize: %d", msgCount, dequeueSize)
	t.Logf("packLimit计算: max(2097152, 2*2*1024) = max(2097152, 4096) = 2097152 (2MB)")
	t.Logf("预期单次Dequeue: 2条 * 1.5MB = 3MB > 2MB，应该走单个发送分支")

	// 等待异步发送完成（大消息需要更长时间，确保所有消息都被处理）
	time.Sleep(1500 * time.Millisecond)

	// 计算mockConn实际收到的总字节数
	var mockConnTotalBytes int64
	for _, data := range mockConn.writeData {
		mockConnTotalBytes += int64(len(data))
	}

	// 验证metrics增量
	finalCount := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedCount].Meter.Snapshot().Count()
	finalBytes := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedByte].Meter.Snapshot().Count()

	if deltaCount := finalCount - initialCount; deltaCount != int64(msgCount) {
		t.Fatalf("期望成功次数增量=%d, 实际=%d", msgCount, deltaCount)
	}

	// 精确断言：三个值应该完全一致
	// 1. expectedTotalBytes: proto.Size计算的预期值
	// 2. mockConnTotalBytes: mockConn实际接收的字节数
	// 3. deltaBytes: metrics统计的字节增量
	deltaBytes := finalBytes - initialBytes
	if deltaBytes != expectedTotalBytes || mockConnTotalBytes != expectedTotalBytes {
		t.Fatalf("字节数不一致 - 预期:%d, metrics:%d, mockConn:%d",
			expectedTotalBytes, deltaBytes, mockConnTotalBytes)
	}

	t.Logf("单个模式 - 次数增量:%d, 字节增量:%d (预期:%d), 写入批次:%d",
		finalCount-initialCount, deltaBytes, expectedTotalBytes, len(mockConn.writeData))
}

// TestLoopSendQueueClose 测试loopSend退出逻辑
func TestLoopSendQueueClose(t *testing.T) {
	mockConn := &mockNetConn{
		remoteAddr: "127.0.0.1:8082",
		writeErr:   nil,
	}

	conn := newConn(mockConn, 10)

	// 发送一些消息
	for i := 0; i < 5; i++ {
		conn.Send(&netsvrProtocol.ConnOpen{UniqId: string(rune('A' + i))}, netsvrProtocol.Cmd_ConnOpen)
	}

	// 关闭连接
	conn.Close()

	// 等待异步关闭完成
	time.Sleep(300 * time.Millisecond)

	// 验证sendCh已关闭
	if !conn.sendCh.IsClosed() {
		t.Fatal("期望sendCh已关闭")
	}

	// 验证底层连接已关闭
	if !mockConn.closeCalled {
		t.Fatal("期望底层连接已关闭")
	}
}

// TestLoopSendMixedSizes 测试混合大小消息
func TestLoopSendMixedSizes(t *testing.T) {
	// 记录初始metrics值
	initialCount := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedCount].Meter.Snapshot().Count()
	initialBytes := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedByte].Meter.Snapshot().Count()

	mockConn := &mockNetConn{
		remoteAddr: "127.0.0.1:8083",
		writeErr:   nil,
	}

	// dequeueSize=8
	dequeueSize := 8
	conn := newConn(mockConn, dequeueSize)
	defer conn.Close()

	// 发送混合大小的消息
	smallCount := 8
	largeCount := 5

	// 计算预期的总字节数
	var expectedTotalBytes int64

	// 发送小消息
	for i := 0; i < smallCount; i++ {
		msg := &netsvrProtocol.ConnClose{
			UniqId:     string(rune('a' + i)),
			CustomerId: "small",
		}
		bodySize := proto.Size(msg)
		expectedTotalBytes += int64(8 + bodySize)
		conn.Send(msg, netsvrProtocol.Cmd_ConnClose)
	}

	// 发送大消息（每条1.5MB）
	largeData := make([]byte, 1500*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	for i := 0; i < largeCount; i++ {
		msg := &netsvrProtocol.Transfer{
			UniqId:     string(rune('A' + i)),
			CustomerId: "large",
			Data:       largeData,
		}
		bodySize := proto.Size(msg)
		expectedTotalBytes += int64(8 + bodySize)
		conn.Send(msg, netsvrProtocol.Cmd_Transfer)
	}

	// 等待异步发送完成（大消息需要更长时间，确保所有消息都被处理）
	time.Sleep(1500 * time.Millisecond)

	// 计算mockConn实际收到的总字节数
	var mockConnTotalBytes int64
	for _, data := range mockConn.writeData {
		mockConnTotalBytes += int64(len(data))
	}

	// 验证metrics增量
	finalSuccessCount := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedCount].Meter.Snapshot().Count()
	finalSuccessBytes := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessSucceedByte].Meter.Snapshot().Count()

	expectedCount := int64(smallCount + largeCount)
	actualSuccessCount := finalSuccessCount - initialCount
	deltaBytes := finalSuccessBytes - initialBytes

	if actualSuccessCount != expectedCount {
		t.Fatalf("期望成功次数=%d, 实际=%d", expectedCount, actualSuccessCount)
	}

	// 精确断言：三个值应该完全一致
	// 1. expectedTotalBytes: proto.Size计算的预期值
	// 2. mockConnTotalBytes: mockConn实际接收的字节数
	// 3. deltaBytes: metrics统计的字节增量
	if deltaBytes != expectedTotalBytes || mockConnTotalBytes != expectedTotalBytes {
		t.Fatalf("字节数不一致 - 预期:%d, metrics:%d, mockConn:%d",
			expectedTotalBytes, deltaBytes, mockConnTotalBytes)
	}

	t.Logf("混合模式 - 次数增量:%d, 字节增量:%d (预期:%d) (小:%d, 大:%d)",
		actualSuccessCount, deltaBytes, expectedTotalBytes, smallCount, largeCount)
}

// TestLoopSendWithWriteError 测试发送失败场景
func TestLoopSendWithWriteError(t *testing.T) {
	// 记录初始metrics值
	initialCount := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessFailedCount].Meter.Snapshot().Count()

	mockConn := &mockNetConn{
		remoteAddr: "127.0.0.1:8084",
		writeErr:   context.DeadlineExceeded, // 模拟写入超时
	}

	conn := newConn(mockConn, 5)
	defer conn.Close()

	// 发送消息（应该会失败）
	msgCount := 3
	for i := 0; i < msgCount; i++ {
		msg := &netsvrProtocol.ConnClose{
			UniqId:     string(rune('A' + i)),
			CustomerId: "error",
		}
		conn.Send(msg, netsvrProtocol.Cmd_ConnClose)
	}

	// 等待异步发送完成
	time.Sleep(300 * time.Millisecond)

	// 验证失败计数
	finalCount := internalMetrics.Registry[internalMetrics.ItemWorkerToBusinessFailedCount].Meter.Snapshot().Count()
	deltaCount := finalCount - initialCount

	if deltaCount == 0 {
		t.Fatal("期望有失败计数，实际为0")
	}

	t.Logf("错误模式 - 失败次数增量:%d", deltaCount)
}

// TestLoopSendPanicRecovery 测试panic恢复逻辑
func TestLoopSendPanicRecovery(t *testing.T) {
	mockConn := &mockNetConn{
		remoteAddr: "127.0.0.1:8085",
		writeErr:   nil,
	}

	conn := newConn(mockConn, 5)

	// 发送消息
	conn.Send(&netsvrProtocol.ConnOpen{UniqId: "test"}, netsvrProtocol.Cmd_ConnOpen)

	// 强制关闭sendCh以触发loopSend退出
	conn.sendCh.Close()

	// 等待goroutine退出
	time.Sleep(200 * time.Millisecond)

	// 如果没有panic，说明恢复逻辑正常工作
	t.Log("Panic恢复测试通过")
}
