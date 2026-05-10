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
	"encoding/binary"
	"fmt"
	"sync"
	"testing"

	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"google.golang.org/protobuf/proto"
)

// TestPacketSet_roundTrip 验证非空消息的 set：proto 编码写入 body、大端 header（总长与 cmd）正确，
// 原地序列化时 bodyFromPool 为 true；reset 后 body 置空且 bodyFromPool 复位，且 body 与原始消息可 Unmarshal 回等价结构。
func TestPacketSet_roundTrip(t *testing.T) {
	msg := &netsvrProtocol.ConnClose{
		UniqId:     "u1",
		CustomerId: "c1",
		Session:    "s1",
		Topics:     []string{"t1", "t2"},
	}
	cmd := netsvrProtocol.Cmd_ConnClose
	pkg := &packet{header: make([]byte, 8)}
	if err := pkg.set(msg, cmd); err != nil {
		t.Fatal(err)
	}
	if !pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool true when marshal reuses pooled buffer")
	}
	wantLen := uint32(len(pkg.body) + 4)
	if got := binary.BigEndian.Uint32(pkg.header[0:4]); got != wantLen {
		t.Fatalf("header length field: got %d want %d", got, wantLen)
	}
	if got := netsvrProtocol.Cmd(binary.BigEndian.Uint32(pkg.header[4:8])); got != cmd {
		t.Fatalf("header cmd field: got %v want %v", got, cmd)
	}
	out := &netsvrProtocol.ConnClose{}
	if err := proto.Unmarshal(pkg.body, out); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(msg, out) {
		t.Fatalf("unmarshaled body mismatch:\ngot  %#v\nwant %#v", out, msg)
	}
	pkg.reset()
	if pkg.body != nil {
		t.Fatal("expected body nil after reset")
	}
	if pkg.bodyFromPool {
		t.Fatal("expected bodyFromPool false after reset")
	}
}

// TestPacketSet_emptyMessage 验证 proto.Size 为 0 的空消息仍能通过 set（池 Get(0) 等边界），
// header 中长度字段与 len(body)+4 一致，且 reset 后 body 被清空。
func TestPacketSet_emptyMessage(t *testing.T) {
	msg := &netsvrProtocol.ConnClose{}
	cmd := netsvrProtocol.Cmd_ConnClose
	pkg := &packet{header: make([]byte, 8)}
	if err := pkg.set(msg, cmd); err != nil {
		t.Fatal(err)
	}
	wantLen := uint32(len(pkg.body) + 4)
	if got := binary.BigEndian.Uint32(pkg.header[0:4]); got != wantLen {
		t.Fatalf("header length field: got %d want %d", got, wantLen)
	}
	pkg.reset()
	if pkg.body != nil {
		t.Fatal("expected body nil after reset")
	}
}

// TestPacketReset_truncatesHeader 验证 reset 将 header 截回固定 8 字节（模拟发送路径上 header 被拉长后的回收），
// 且当 bodyFromPool 为 false 时不会对非 byteslice 池的 body 调用 Put（仅用字面量 body 避免污染池）。
func TestPacketReset_truncatesHeader(t *testing.T) {
	pkg := &packet{
		header:       append(make([]byte, 8), []byte{0xab, 0xcd}...),
		body:         []byte("not-from-pool"),
		bodyFromPool: false,
	}
	pkg.reset()
	if len(pkg.header) != 8 {
		t.Fatalf("header len: got %d want 8", len(pkg.header))
	}
	if string(pkg.body) != "" {
		t.Fatalf("body: got %q want empty", pkg.body)
	}
}

// TestPacketSet_cycleAfterReset 验证同一 packet 上 set → reset → set 的复用流程：第二次编码覆盖第一次，
// Unmarshal 得到第二条消息内容，避免残留上一包数据。
func TestPacketSet_cycleAfterReset(t *testing.T) {
	first := &netsvrProtocol.ConnClose{UniqId: "a"}
	second := &netsvrProtocol.ConnClose{UniqId: "bbbb"}
	pkg := &packet{header: make([]byte, 8)}

	if err := pkg.set(first, netsvrProtocol.Cmd_ConnClose); err != nil {
		t.Fatal(err)
	}
	pkg.reset()

	if err := pkg.set(second, netsvrProtocol.Cmd_ConnOpen); err != nil {
		t.Fatal(err)
	}
	if proto.Equal(first, second) {
		t.Fatal("messages should differ")
	}
	out := &netsvrProtocol.ConnClose{}
	if err := proto.Unmarshal(pkg.body, out); err != nil {
		t.Fatal(err)
	}
	if out.UniqId != "bbbb" {
		t.Fatalf("second round body: got uniqId %q", out.UniqId)
	}
	pkg.reset()
}

// TestPacketPoolPutInvokesReset 验证全局 packetObjPool.Put 会调用 reset：再次 Get 得到的对象 header 长度为 8、
// body 为空且 bodyFromPool 为 false，保证回池时状态被清理干净，可供下一轮 Send 复用。
func TestPacketPoolPutInvokesReset(t *testing.T) {
	pkg := packetObjPool.Get()
	pkg.header = append(pkg.header, 9, 9, 9)
	msg := &netsvrProtocol.ConnClose{UniqId: "pool"}
	if err := pkg.set(msg, netsvrProtocol.Cmd_ConnClose); err != nil {
		t.Fatal(err)
	}
	packetObjPool.Put(pkg)

	pkg2 := packetObjPool.Get()
	if len(pkg2.header) != 8 {
		t.Fatalf("after Put/Get header len: got %d want 8", len(pkg2.header))
	}
	if pkg2.body != nil || pkg2.bodyFromPool {
		t.Fatal("expected clean packet from pool")
	}
}

// TestPacketSet_largeMessage 验证较大 payload 的 set：header 与 wire 长度一致、body 可 Unmarshal 回等价消息，
// reset 后 body 清空。在当前 byteslice 分级下该消息通常仍原地序列化（bodyFromPool 多为 true），仅作日志不强制断言以免与实现细节耦合。
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
	pkg := &packet{header: make([]byte, 8)}

	if err := pkg.set(msg, cmd); err != nil {
		t.Fatal(err)
	}

	// 验证 header
	wantLen := uint32(len(pkg.body) + 4)
	if got := binary.BigEndian.Uint32(pkg.header[0:4]); got != wantLen {
		t.Fatalf("header length: got %d want %d", got, wantLen)
	}
	if got := netsvrProtocol.Cmd(binary.BigEndian.Uint32(pkg.header[4:8])); got != cmd {
		t.Fatalf("header cmd: got %v want %v", got, cmd)
	}

	t.Logf("bodyFromPool: %v, body size: %d", pkg.bodyFromPool, len(pkg.body))

	// 验证数据完整性
	out := &netsvrProtocol.Transfer{}
	if err := proto.Unmarshal(pkg.body, out); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(msg, out) {
		t.Fatal("large message unmarshaled mismatch")
	}

	pkg.reset()
	if pkg.body != nil {
		t.Fatal("expected body nil after reset")
	}
}

// TestPacketSet_headerFields 验证 header 中长度字段和 cmd 字段的精确性，
// 确保长度 = len(body) + 4（包含 cmd 的 4 字节）。
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
			pkg := &packet{header: make([]byte, 8)}
			if err := pkg.set(tt.msg, tt.cmd); err != nil {
				t.Fatal(err)
			}

			// 验证长度字段：应该是 body 长度 + 4（cmd 占用 4 字节）
			wantLen := uint32(len(pkg.body) + 4)
			gotLen := binary.BigEndian.Uint32(pkg.header[0:4])
			if gotLen != wantLen {
				t.Errorf("length field: got %d want %d (body=%d)", gotLen, wantLen, len(pkg.body))
			}

			// 验证 cmd 字段
			gotCmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(pkg.header[4:8]))
			if gotCmd != tt.cmd {
				t.Errorf("cmd field: got %v want %v", gotCmd, tt.cmd)
			}

			pkg.reset()
		})
	}
}

// TestPacketSet_concurrent 验证多 goroutine 对 packetObjPool 的 Get / set / Put：各自持有独立 packet，
// 不应互相覆盖 UniqId。错误在子 goroutine 中收集，由测试 goroutine 统一 t.Error，符合 testing 用法。
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
				if err := proto.Unmarshal(pkg.body, out); err != nil {
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

// BenchmarkPacketSet_WithPool 基准测试：使用 packetObjPool 进行序列化（包含 Get / set / Put，含 8 字节 header 写入）。
// 对照组 BenchmarkPacketSet_NativeMarshal 仅测 proto.Marshal，不包含包头，二者并非完全等价工作量，但可对比「线上 Send 路径」与「裸序列化」的量级。
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

// BenchmarkPacketSet_LargeMessage_WithPool 基准测试：约 10KiB payload 时 Get / set / Put 全路径（byteslice 复用 + 包头）。
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
