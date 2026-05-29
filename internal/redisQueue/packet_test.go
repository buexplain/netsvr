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

// Package redisQueue Redis队列
package redisQueue

import (
	"encoding/binary"
	"fmt"
	"sync"
	"testing"

	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"google.golang.org/protobuf/proto"
)

// TestPacketSet_roundTrip 验证非空消息的 set：
// proto 编码写入 message[4:]、大端 cmd 写入 message[0:4] 正确，
// 原地序列化时 bodyFromPool 为 true；
// reset 后 message 置空且 bodyFromPool 复位；
// message[4:] 与原始消息可 Unmarshal 回等价结构。
func TestPacketSet_roundTrip(t *testing.T) {
	msg := &netsvrProtocol.ConnClose{
		UniqId:     "u1",
		CustomerId: "c1",
		Session:    "s1",
		Topics:     []string{"t1", "t2"},
	}
	cmd := netsvrProtocol.Cmd_ConnClose
	pkg := &packet{}
	if err := pkg.set(msg, cmd); err != nil {
		t.Fatal(err)
	}
	if !pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool true when marshal reuses pooled buffer")
	}
	// message[0:4] 是 cmd（大端序 uint32）
	if got := netsvrProtocol.Cmd(binary.BigEndian.Uint32(pkg.message[0:4])); got != cmd {
		t.Fatalf("cmd field: got %v want %v", got, cmd)
	}
	out := &netsvrProtocol.ConnClose{}
	if err := proto.Unmarshal(pkg.message[4:], out); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(msg, out) {
		t.Fatalf("unmarshaled body mismatch:\ngot  %#v\nwant %#v", out, msg)
	}
	pkg.reset()
	if pkg.message != nil {
		t.Fatal("expected message nil after reset")
	}
	if pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool false after reset")
	}
}

// TestPacketSet_emptyMessage 验证 proto.Size 为 0 的空消息仍能通过 set（make([]byte, 4) 路径），
// message 中 cmd 字段正确，bodyFromPool 为 false，
// 且 reset 后 message 被清空。
func TestPacketSet_emptyMessage(t *testing.T) {
	msg := &netsvrProtocol.ConnClose{}
	cmd := netsvrProtocol.Cmd_ConnClose
	pkg := &packet{}
	if err := pkg.set(msg, cmd); err != nil {
		t.Fatal(err)
	}
	if pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool false for empty message (make path)")
	}
	if got := netsvrProtocol.Cmd(binary.BigEndian.Uint32(pkg.message[0:4])); got != cmd {
		t.Fatalf("cmd field: got %v want %v", got, cmd)
	}
	// 空消息的 body 为 0 字节，所以 message[4:] 应为空切片
	if len(pkg.message[4:]) != 0 {
		t.Fatalf("expected empty body for empty message, got len=%d", len(pkg.message[4:]))
	}
	pkg.reset()
	if pkg.message != nil {
		t.Fatal("expected message nil after reset")
	}
}

// TestPacketReset_bodyFromPoolTrue 验证 reset 在 bodyFromPool 为 true 时归还 message 到 byteslice 池，
// 之后 message 被置空、bodyFromPool 复位为 false。
func TestPacketReset_bodyFromPoolTrue(t *testing.T) {
	msg := &netsvrProtocol.Transfer{
		UniqId:     "test-reset-pool",
		CustomerId: "c-reset",
		Data:       []byte("some payload data"),
	}
	pkg := &packet{}
	if err := pkg.set(msg, netsvrProtocol.Cmd_Transfer); err != nil {
		t.Fatal(err)
	}
	if !pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool true before reset")
	}
	pkg.reset()
	if pkg.message != nil {
		t.Fatal("expected message nil after reset")
	}
	if pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool false after reset")
	}
}

// TestPacketReset_bodyFromPoolFalse 验证 reset 在 bodyFromPool 为 false 时不会调用 byteslice.Put，
// 仅将 message 置空。使用空消息（size <= 0 路径）确保 bodyFromPool = false。
func TestPacketReset_bodyFromPoolFalse(t *testing.T) {
	msg := &netsvrProtocol.ConnClose{}
	pkg := &packet{}
	if err := pkg.set(msg, netsvrProtocol.Cmd_ConnClose); err != nil {
		t.Fatal(err)
	}
	if pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool false for empty message")
	}
	pkg.reset()
	if pkg.message != nil {
		t.Fatal("expected message nil after reset when bodyFromPool is false")
	}
	if pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool false after reset")
	}
}

// TestPacketReset_nilMessage 验证 reset 在 message 为 nil 时的安全性，
// 不会 panic 或错误调用 byteslice.Put。
func TestPacketReset_nilMessage(t *testing.T) {
	pkg := &packet{message: nil, bodyFromPool: false}
	pkg.reset()
	if pkg.message != nil {
		t.Fatal("expected message nil after reset on nil message")
	}
	if pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool false after reset")
	}
}

// TestPacketSet_cycleAfterReset 验证同一 packet 上 set → reset → set 的复用流程：
// 第二次编码覆盖第一次，Unmarshal 得到第二条消息内容，避免残留上一包数据。
func TestPacketSet_cycleAfterReset(t *testing.T) {
	first := &netsvrProtocol.ConnClose{UniqId: "a"}
	second := &netsvrProtocol.ConnClose{UniqId: "bbbb"}
	pkg := &packet{}

	if err := pkg.set(first, netsvrProtocol.Cmd_ConnClose); err != nil {
		t.Fatal(err)
	}
	out1 := &netsvrProtocol.ConnClose{}
	if err := proto.Unmarshal(pkg.message[4:], out1); err != nil {
		t.Fatal(err)
	}
	if out1.UniqId != first.UniqId {
		t.Fatalf("first round: got uniqId %q want %q", out1.UniqId, first.UniqId)
	}
	pkg.reset()

	if err := pkg.set(second, netsvrProtocol.Cmd_ConnOpen); err != nil {
		t.Fatal(err)
	}
	if proto.Equal(first, second) {
		t.Fatal("messages should differ")
	}
	out2 := &netsvrProtocol.ConnClose{}
	if err := proto.Unmarshal(pkg.message[4:], out2); err != nil {
		t.Fatal(err)
	}
	if out2.UniqId != second.UniqId {
		t.Fatalf("second round body: got uniqId %q want %q", out2.UniqId, second.UniqId)
	}
	// 验证第二次 set 的 cmd 是 ConnOpen
	if gotCmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(pkg.message[0:4])); gotCmd != netsvrProtocol.Cmd_ConnOpen {
		t.Fatalf("second round cmd: got %v want %v", gotCmd, netsvrProtocol.Cmd_ConnOpen)
	}
	pkg.reset()
}

// TestPacketSet_headerFields 验证 message 中 cmd 字段的精确性，
// 确保数据完整性：message[4:] 可 Unmarshal 回等价消息。
func TestPacketSet_headerFields(t *testing.T) {
	tests := []struct {
		name string
		msg  proto.Message
		cmd  netsvrProtocol.Cmd
	}{
		{
			name: "ConnOpen",
			msg: &netsvrProtocol.ConnOpen{
				UniqId:     "test-uniq",
				RawQuery:   "?key=value",
				RemoteAddr: "127.0.0.1:8080",
			},
			cmd: netsvrProtocol.Cmd_ConnOpen,
		},
		{
			name: "Transfer with data",
			msg: &netsvrProtocol.Transfer{
				UniqId:     "u1",
				CustomerId: "c1",
				Data:       []byte("hello world"),
			},
			cmd: netsvrProtocol.Cmd_Transfer,
		},
		{
			name: "ConnClose with topics",
			msg: &netsvrProtocol.ConnClose{
				UniqId:     "u2",
				CustomerId: "c2",
				Session:    "s2",
				Topics:     []string{"t1", "t2", "t3"},
			},
			cmd: netsvrProtocol.Cmd_ConnClose,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &packet{}
			if err := pkg.set(tt.msg, tt.cmd); err != nil {
				t.Fatal(err)
			}

			// 验证 cmd 字段
			gotCmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(pkg.message[0:4]))
			if gotCmd != tt.cmd {
				t.Errorf("cmd field: got %v want %v", gotCmd, tt.cmd)
			}

			// 验证数据完整性：message[4:] 可 Unmarshal 回等价消息
			out := proto.Clone(tt.msg)
			proto.Reset(out)
			if err := proto.Unmarshal(pkg.message[4:], out); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if !proto.Equal(tt.msg, out) {
				t.Errorf("body mismatch: proto.Equal returned false")
			}

			pkg.reset()
		})
	}
}

// TestPacketSet_headerNotCorruptsBody 验证 cmd 写入不会破坏 proto body 数据：
// 先 Marshal 得到纯净 body，再 set 写入 cmd，比较 message[4:] 与纯净 body 完全一致。
func TestPacketSet_headerNotCorruptsBody(t *testing.T) {
	msg := &netsvrProtocol.Transfer{
		UniqId:     "corruption-test",
		CustomerId: "c1",
		Session:    "s1",
		Data:       []byte("this is the original body data that must not be corrupted"),
	}
	// 先获取纯净的 proto body 数据
	pureBody, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	pkg := &packet{}
	if err := pkg.set(msg, netsvrProtocol.Cmd_Transfer); err != nil {
		t.Fatal(err)
	}

	// message[4:] 应与纯净 body 完全一致
	actualBody := pkg.message[4:]
	if len(actualBody) != len(pureBody) {
		t.Fatalf("body length mismatch: got %d want %d", len(actualBody), len(pureBody))
	}
	for i := range pureBody {
		if actualBody[i] != pureBody[i] {
			t.Fatalf("body byte mismatch at index %d: got %d want %d", i, actualBody[i], pureBody[i])
		}
	}
	pkg.reset()
}

// TestPacketPoolPutInvokesReset 验证全局 packetObjPool.Put 会调用 reset：
// 再次 Get 得到的对象 message 为空且 bodyFromPool 为 false，
// 保证回池时状态被清理干净，可供下一轮 Send 复用。
func TestPacketPoolPutInvokesReset(t *testing.T) {
	pkg := packetObjPool.Get()
	msg := &netsvrProtocol.ConnClose{UniqId: "pool"}
	if err := pkg.set(msg, netsvrProtocol.Cmd_ConnClose); err != nil {
		t.Fatal(err)
	}
	packetObjPool.Put(pkg)

	pkg2 := packetObjPool.Get()
	if pkg2.message != nil || pkg2.bodyFromPool {
		t.Fatal("expected clean packet from pool after Put/Get")
	}
}

// TestPacketPoolGetReturnsNewObject 验证首次从 packetObjPool.Get 得到的对象是全新且干净的。
func TestPacketPoolGetReturnsNewObject(t *testing.T) {
	pkg := packetObjPool.Get()
	if pkg.message != nil {
		t.Fatal("expected nil message from fresh pool Get")
	}
	if pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool false from fresh pool Get")
	}
	packetObjPool.Put(pkg)
}

// TestPacketSet_largeMessage 验证较大 payload 的 set：
// message 中 cmd 字段正确、body 可 Unmarshal 回等价消息，
// reset 后 message 清空。
func TestPacketSet_largeMessage(t *testing.T) {
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	msg := &netsvrProtocol.Transfer{
		UniqId:     "large-test",
		CustomerId: "customer-123",
		Session:    "session-456",
		Topics:     []string{"topic1", "topic2", "topic3"},
		Data:       largeData,
	}
	cmd := netsvrProtocol.Cmd_Transfer
	pkg := &packet{}

	if err := pkg.set(msg, cmd); err != nil {
		t.Fatal(err)
	}

	// 验证 cmd 字段
	if got := netsvrProtocol.Cmd(binary.BigEndian.Uint32(pkg.message[0:4])); got != cmd {
		t.Fatalf("cmd field: got %v want %v", got, cmd)
	}

	t.Logf("bodyFromPool: %v, message size: %d", pkg.bodyFromPool, len(pkg.message))

	// 验证数据完整性
	out := &netsvrProtocol.Transfer{}
	if err := proto.Unmarshal(pkg.message[4:], out); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(msg, out) {
		t.Fatal("large message unmarshaled mismatch")
	}

	pkg.reset()
	if pkg.message != nil {
		t.Fatal("expected message nil after reset")
	}
}

// TestPacketSet_smallMessage 验证很小的 proto 消息（size < 4）的 set 不会 panic：
// message[0:4] 可以安全写入 cmd，message[4:] 可正确 Unmarshal。
func TestPacketSet_smallMessage(t *testing.T) {
	msg := &netsvrProtocol.ConnClose{UniqId: "x"} // 很小的消息
	cmd := netsvrProtocol.Cmd_ConnClose
	pkg := &packet{}

	if err := pkg.set(msg, cmd); err != nil {
		t.Fatal(err)
	}

	// 验证 cmd 字段可安全访问
	if got := netsvrProtocol.Cmd(binary.BigEndian.Uint32(pkg.message[0:4])); got != cmd {
		t.Fatalf("cmd field: got %v want %v", got, cmd)
	}

	// 验证 body 可 Unmarshal
	out := &netsvrProtocol.ConnClose{}
	if err := proto.Unmarshal(pkg.message[4:], out); err != nil {
		t.Fatal(err)
	}
	if out.UniqId != "x" {
		t.Fatalf("uniqId: got %q want %q", out.UniqId, "x")
	}

	pkg.reset()
}

// TestPacketSet_concurrent 验证多 goroutine 对 packetObjPool 的 Get / set / Put：
// 各自持有独立 packet，不应互相覆盖 UniqId。
// 错误在子 goroutine 中收集，由测试 goroutine 统一 t.Error。
func TestPacketSet_concurrent(t *testing.T) {
	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				pkg := packetObjPool.Get()
				msg := &netsvrProtocol.ConnClose{
					UniqId:     fmt.Sprintf("g%d-i%d", id, i),
					CustomerId: fmt.Sprintf("customer-%d", id),
				}
				if err := pkg.set(msg, netsvrProtocol.Cmd_ConnClose); err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("goroutine %d iteration %d set: %w", id, i, err))
					mu.Unlock()
					packetObjPool.Put(pkg)
					return
				}
				out := &netsvrProtocol.ConnClose{}
				if err := proto.Unmarshal(pkg.message[4:], out); err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("goroutine %d iteration %d unmarshal: %w", id, i, err))
					mu.Unlock()
					packetObjPool.Put(pkg)
					return
				}
				if out.UniqId != msg.UniqId {
					mu.Lock()
					errs = append(errs, fmt.Errorf("goroutine %d iteration %d: got UniqId %q want %q", id, i, out.UniqId, msg.UniqId))
					mu.Unlock()
					packetObjPool.Put(pkg)
					return
				}
				packetObjPool.Put(pkg)
			}
		}(g)
	}

	wg.Wait()
	for _, err := range errs {
		t.Error(err)
	}
}

// BenchmarkPacketSet_WithPool 基准测试：使用 packetObjPool 进行序列化
// （包含 Get / set / Put，含 4 字节 cmd 前缀写入）。
func BenchmarkPacketSet_WithPool(b *testing.B) {
	msg := &netsvrProtocol.Transfer{
		UniqId:     "bench-uniq-id",
		CustomerId: "bench-customer",
		Session:    "bench-session",
		Topics:     []string{"topic1", "topic2", "topic3"},
		Data:       []byte("hello world test data for benchmark"),
	}
	cmd := netsvrProtocol.Cmd_Transfer
	b.SetBytes(int64(proto.Size(msg)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pkg := packetObjPool.Get()
		if err := pkg.set(msg, cmd); err != nil {
			b.Fatal(err)
		}
		packetObjPool.Put(pkg)
	}
}

// BenchmarkPacketSet_NativeMarshal 基准测试：每轮仅 proto.Marshal（每轮 1 次堆分配），作分配与耗时的对照基线。
func BenchmarkPacketSet_NativeMarshal(b *testing.B) {
	msg := &netsvrProtocol.Transfer{
		UniqId:     "bench-uniq-id",
		CustomerId: "bench-customer",
		Session:    "bench-session",
		Topics:     []string{"topic1", "topic2", "topic3"},
		Data:       []byte("hello world test data for benchmark"),
	}
	b.SetBytes(int64(proto.Size(msg)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

// BenchmarkPacketSet_LargeMessage_WithPool 基准测试：约 10KiB payload 时 Get / set / Put 全路径。
func BenchmarkPacketSet_LargeMessage_WithPool(b *testing.B) {
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	msg := &netsvrProtocol.Transfer{
		UniqId:     "bench-large",
		CustomerId: "bench-customer",
		Session:    "bench-session",
		Topics:     []string{"topic1", "topic2"},
		Data:       largeData,
	}
	cmd := netsvrProtocol.Cmd_Transfer
	b.SetBytes(int64(proto.Size(msg)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pkg := packetObjPool.Get()
		if err := pkg.set(msg, cmd); err != nil {
			b.Fatal(err)
		}
		packetObjPool.Put(pkg)
	}
}

// BenchmarkPacketSet_LargeMessage_NativeMarshal 基准测试：同 payload 下每轮 proto.Marshal 的分配与耗时（约 10KiB/轮分配）。
func BenchmarkPacketSet_LargeMessage_NativeMarshal(b *testing.B) {
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	msg := &netsvrProtocol.Transfer{
		UniqId:     "bench-large",
		CustomerId: "bench-customer",
		Session:    "bench-session",
		Topics:     []string{"topic1", "topic2"},
		Data:       largeData,
	}
	b.SetBytes(int64(proto.Size(msg)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}
