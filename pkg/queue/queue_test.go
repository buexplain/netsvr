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

package queue

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// TestPadding 测试缓存行填充
func TestPadding(t *testing.T) {
	q := New[int](4)

	// 检查 head 和 tail 的偏移差
	headOffset := unsafe.Offsetof(q.head)
	tailOffset := unsafe.Offsetof(q.tail)

	t.Logf("head offset: %d, tail offset: %d, diff: %d bytes",
		headOffset, tailOffset, tailOffset-headOffset)

	if tailOffset-headOffset < cacheLineSize {
		t.Errorf("head and tail too close: %d bytes (should be >= %d)",
			tailOffset-headOffset, cacheLineSize)
	}

	// 检查 tail 到 mask 的间距
	maskOffset := unsafe.Offsetof(q.mask)
	t.Logf("tail to mask: %d bytes", maskOffset-tailOffset)

	// tail 应该有完整的 cache line 填充
	expectedMinMaskOffset := tailOffset + cacheLineSize
	if maskOffset < expectedMinMaskOffset {
		t.Errorf("mask too close to tail: %d bytes (should be >= %d)",
			maskOffset-tailOffset, cacheLineSize)
	}

	// 检查 slot stride
	slotSize := unsafe.Sizeof(slot[int]{})
	t.Logf("slot size: %d, stride: %d", slotSize, q.slots.stride)

	if q.slots.stride < cacheLineSize {
		t.Errorf("stride too small: %d < %d", q.slots.stride, cacheLineSize)
	}

}

// TestPaddingDifferentTypes 测试不同类型参数的缓存行填充
func TestPaddingDifferentTypes(t *testing.T) {
	testCases := []struct {
		name     string
		capacity int
	}{
		{"int", 4},
		{"string", 4},
		{"[]byte", 4},
		{"[2][]byte", 4}, // 项目实际使用的类型
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stride uintptr
			var actualSlotSize uintptr

			switch tc.name {
			case "int":
				q := New[int](tc.capacity)
				stride = q.slots.stride
				actualSlotSize = unsafe.Sizeof(slot[int]{})
				t.Logf("int - slot size: %d, stride: %d",
					actualSlotSize, stride)

			case "string":
				q := New[string](tc.capacity)
				stride = q.slots.stride
				actualSlotSize = unsafe.Sizeof(slot[string]{})
				t.Logf("string - slot size: %d, stride: %d",
					actualSlotSize, stride)

			case "[]byte":
				q := New[[]byte](tc.capacity)
				stride = q.slots.stride
				actualSlotSize = unsafe.Sizeof(slot[[]byte]{})
				t.Logf("[]byte - slot size: %d, stride: %d",
					actualSlotSize, stride)

			case "[2][]byte":
				q := New[[2][]byte](tc.capacity)
				stride = q.slots.stride
				actualSlotSize = unsafe.Sizeof(slot[[2][]byte]{})
				t.Logf("[2][]byte - slot size: %d, stride: %d",
					actualSlotSize, stride)
			}

			// 验证 stride >= cacheLineSize
			if stride < cacheLineSize {
				t.Errorf("%s: stride too small: %d < %d",
					tc.name, stride, cacheLineSize)
			}

			// 验证 stride >= 实际 slot 大小
			if stride < actualSlotSize {
				t.Errorf("%s: stride smaller than slot: %d < %d",
					tc.name, stride, actualSlotSize)
			}

			// 记录内存利用率
			utilization := float64(actualSlotSize) / float64(stride) * 100
			t.Logf("%s - memory utilization: %.2f%% (%d/%d bytes)",
				tc.name, utilization, actualSlotSize, stride)
		})
	}
}

// TestPaddingStructureLayout 测试结构体内存布局
func TestPaddingStructureLayout(t *testing.T) {
	type testStruct struct {
		head   atomic.Uint64
		_      [cacheLineSize - unsafe.Sizeof(atomic.Uint64{})]byte
		tail   atomic.Uint64
		_      [cacheLineSize - unsafe.Sizeof(atomic.Uint64{})]byte
		mask   uint64
		closed atomic.Bool
		_      [cacheLineSize - unsafe.Sizeof(uint64(0)) - unsafe.Sizeof(atomic.Bool{})]byte
	}

	headOffset := unsafe.Offsetof(testStruct{}.head)
	tailOffset := unsafe.Offsetof(testStruct{}.tail)
	maskOffset := unsafe.Offsetof(testStruct{}.mask)

	t.Logf("testStruct layout: head=%d, tail=%d, mask=%d",
		headOffset, tailOffset, maskOffset)

	// 验证 head 和 tail 间隔
	if tailOffset-headOffset < cacheLineSize {
		t.Errorf("head/tail spacing insufficient: %d < %d",
			tailOffset-headOffset, cacheLineSize)
	}

	// 验证 tail 和 mask 间隔
	if maskOffset-tailOffset < cacheLineSize {
		t.Errorf("tail/mask spacing insufficient: %d < %d",
			maskOffset-tailOffset, cacheLineSize)
	}

	t.Logf("total struct size: %d bytes", unsafe.Sizeof(testStruct{}))
}

// TestBasicEnqueueDequeue 测试基本的入队和出队操作
func TestBasicEnqueueDequeue(t *testing.T) {
	q := New[int](4)
	result := make([]int, 10)

	// 入队 3 个元素
	if !q.Enqueue(1) {
		t.Fatal("入队 1 失败")
	}
	if !q.Enqueue(2) {
		t.Fatal("入队 2 失败")
	}
	if !q.Enqueue(3) {
		t.Fatal("入队 3 失败")
	}

	// 出队验证
	count := q.Dequeue(result)
	if count != 3 {
		t.Fatalf("期望出队 3 个元素，实际出队%d个", count)
	}
	if result[0] != 1 || result[1] != 2 || result[2] != 3 {
		t.Fatalf("出队数据不正确：%v", result[:3])
	}
}

// TestSequenceCycle 测试序列号循环（核心修复验证）
// 这是验证修复是否正确的关键测试
func TestSequenceCycle(t *testing.T) {
	q := New[int](4)
	result := make([]int, 2)

	// 第一轮：入队 2 个，出队 2 个
	q.Enqueue(10)
	q.Enqueue(20)
	q.Dequeue(result)

	// 第二轮：再次入队 2 个，出队 2 个
	q.Enqueue(30)
	q.Enqueue(40)
	count := q.Dequeue(result)

	if count != 2 {
		t.Fatalf("第二轮期望出队 2 个，实际%d", count)
	}
	if result[0] != 30 || result[1] != 40 {
		t.Fatalf("第二轮数据不正确：%v", result[:2])
	}

	// 第三轮：再次验证循环
	q.Enqueue(50)
	q.Enqueue(60)
	count = q.Dequeue(result)

	if count != 2 {
		t.Fatalf("第三轮期望出队 2 个，实际%d", count)
	}
	if result[0] != 50 || result[1] != 60 {
		t.Fatalf("第三轮数据所在确：%v", result[:2])
	}
}

// TestFullCycle 测试完整循环（填满队列，全部出队，再填满）
func TestFullCycle(t *testing.T) {
	q := New[int](8)
	result := make([]int, 8)

	// 第一轮：填满队列
	for i := 0; i < 8; i++ {
		if !q.Enqueue(i) {
			t.Fatalf("第一轮入队%d失败", i)
		}
	}

	// 出队全部
	count := q.Dequeue(result)
	if count != 8 {
		t.Fatalf("第一轮期望出队 8 个，实际%d", count)
	}

	// 第二轮：再次填满
	for i := 10; i < 18; i++ {
		if !q.Enqueue(i) {
			t.Fatalf("第二轮入队%d失败", i)
		}
	}

	// 出队验证
	count = q.Dequeue(result)
	if count != 8 {
		t.Fatalf("第二轮期望出队 8 个，实际%d", count)
	}

	for i := 0; i < 8; i++ {
		if result[i] != 10+i {
			t.Fatalf("第二轮数据 [%d] 期望%d，实际%d", i, 10+i, result[i])
		}
	}
}

// TestConcurrentEnqueue 测试多生产者并发入队
func TestConcurrentEnqueue(t *testing.T) {
	q := New[int](1024)
	wg := &sync.WaitGroup{}
	total := 1000

	// 启动 10 个生产者
	for p := 0; p < 10; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			for i := 0; i < total/10; i++ {
				item := producerID*100 + i
				// 添加重试机制，队列满时让出 CPU
				for !q.Enqueue(item) {
					runtime.Gosched()
				}
			}
		}(p)
	}
	wg.Wait()

	// 验证所有元素都被入队
	if q.Len() != total {
		t.Fatalf("期望队列长度%d，实际%d", total, q.Len())
	}
}

// TestConcurrentEnqueueDequeue 测试多生产者单消费者并发
func TestConcurrentEnqueueDequeue(t *testing.T) {
	q := New[int](1024)
	var producerWg = &sync.WaitGroup{}
	var consumerWg = &sync.WaitGroup{}
	total := 1000

	// 启动消费者
	dequeued := make(chan int, total*2) // 增大缓冲区

	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		result := make([]int, 100)
		for {
			count := q.Dequeue(result)
			if count == 0 {
				close(dequeued)
				return
			}
			for i := 0; i < count; i++ {
				dequeued <- result[i]
			}
		}
	}()

	// 启动生产者
	for p := 0; p < 10; p++ {
		producerWg.Add(1)
		go func(producerID int) {
			defer producerWg.Done()
			for i := 0; i < total/10; i++ {
				item := producerID*100 + i
				for !q.Enqueue(item) {
					runtime.Gosched()
				}
			}
		}(p)
	}

	// 等待所有生产者完成
	producerWg.Wait()

	// 关闭队列
	q.Close()

	// 等待消费者完成
	consumerWg.Wait()

	// 验证所有元素都被正确消费
	seen := make(map[int]bool)
	for item := range dequeued {
		if seen[item] {
			t.Fatalf("重复元素：%d", item)
		}
		seen[item] = true
	}

	if len(seen) != total {
		t.Fatalf("期望消费%d个唯一元素，实际%d", total, len(seen))
	}
}

// TestConcurrentFullCycle 测试并发场景下的完整循环（
// 使用 MPSC 队列：多生产者单消费者
func TestConcurrentFullCycle(t *testing.T) {
	q := New[int](64)
	wg := &sync.WaitGroup{}
	totalRounds := 10
	itemsPerRound := 50

	// 启动单个消费者，持续消费所有轮次的数据
	dequeued := make([]int, 0, totalRounds*itemsPerRound)
	consumerDone := make(chan struct{})

	go func() {
		defer func() { consumerDone <- struct{}{} }()
		result := make([]int, 50)
		for {
			count := q.Dequeue(result)
			if count > 0 {
				dequeued = append(dequeued, result[:count]...)
			} else {
				return
			}
		}
	}()

	// 多轮生产者
	for round := 0; round < totalRounds; round++ {
		wg.Add(1)
		go func(r int) {
			defer wg.Done()
			for i := 0; i < itemsPerRound; i++ {
				for !q.Enqueue(r*100 + i) {
					runtime.Gosched()
				}
			}
		}(round)
	}

	// 等待所有生产者完成
	wg.Wait()

	// 关闭队列
	q.Close()

	// 等待消费者完成
	<-consumerDone

	// 验证消费了 500 个元素
	if len(dequeued) != totalRounds*itemsPerRound {
		t.Fatalf("期望消费%d个元素，实际%d", totalRounds*itemsPerRound, len(dequeued))
	}
}

// TestEmptyQueue 测试空队列的出队
func TestEmptyQueue(t *testing.T) {
	q := New[int](4)
	result := make([]int, 10)
	go func() {
		time.Sleep(time.Millisecond * 100)
		q.Close()
	}()
	count := q.Dequeue(result)
	if count != 0 {
		t.Fatalf("空队列出队期望 0，实际%d", count)
	}
}

// TestFullQueue 测试满队列的入队
func TestFullQueue(t *testing.T) {
	q := New[int](4)

	// 填满队列
	for i := 0; i < 4; i++ {
		if !q.Enqueue(i) {
			t.Fatalf("入队%d失败", i)
		}
	}

	// 第 5 个入队应该失败
	if q.Enqueue(99) {
		t.Fatal("满队列入队应该失败")
	}
}

// TestTryEnqueue 测试 TryEnqueue 单次尝试
func TestTryEnqueue(t *testing.T) {
	q := New[int](4)

	// 成功入队
	if !q.TryEnqueue(1) {
		t.Fatal("TryEnqueue 应该成功")
	}

	// 验证数据
	result := make([]int, 1)
	count := q.Dequeue(result)
	if count != 1 || result[0] != 1 {
		t.Fatalf("数据不正确：count=%d, result[0]=%d", count, result[0])
	}

	// 测试 TryEnqueue 单次尝试
	// 填满队列后，TryEnqueue 应该返回 false
	for i := 0; i < 4; i++ {
		q.Enqueue(i)
	}
	// 队列已满，TryEnqueue 应该立即返回 false
	if q.TryEnqueue(99) {
		t.Fatal("TryEnqueue 在队列满时应该返回 false")
	}
}

// TestTryEnqueueCAS 测试 TryEnqueue CAS
func TestTryEnqueueCAS(t *testing.T) {
	var count int32 = 0
	countPtr := &count
	q := New[int](4)
	wg := &sync.WaitGroup{}
	startSignal := make(chan struct{})
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int, countPtr *int32) {
			defer wg.Done()
			<-startSignal
			if q.TryEnqueue(i) {
				atomic.AddInt32(countPtr, 1)
			}
		}(i, countPtr)
	}
	time.Sleep(300 * time.Millisecond) // 确保所有协程都已启动并阻塞
	close(startSignal)                 // 广播信号，所有协程同时开始
	wg.Wait()
	if q.Len() != int(*countPtr) {
		t.Fatalf("TryEnqueue CAS 错误，期望%d，实际%d", q.Len(), count)
	}
}

// TestEnqueueCAS 测试 Enqueue CAS
func TestEnqueueCAS(t *testing.T) {
	q := New[int](4)
	wg := &sync.WaitGroup{}
	startSignal := make(chan struct{})
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-startSignal
			q.Enqueue(i)
		}(i)
	}
	time.Sleep(300 * time.Millisecond) // 确保所有协程都已启动并阻塞
	close(startSignal)                 // 广播信号，所有协程同时开始
	wg.Wait()
}

// TestCloseQueue 测试关闭队列
func TestCloseQueue(t *testing.T) {
	q := New[int](4)

	// 正常入队
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)
	q.Enqueue(4)

	// 关闭队列
	q.Close()

	// 关闭后入队应该失败
	if q.Enqueue(5) {
		t.Fatal("关闭后入队应该失败")
	}

	if q.TryEnqueue(6) {
		t.Fatal("关闭后入队应该失败")
	}

	// 已入队的数据仍可消费
	result := make([]int, 2)
	//第一轮消费
	count := q.Dequeue(result)
	if count != 2 || result[0] != 1 || result[1] != 2 {
		t.Fatalf("关闭后消费数据不正确：%v", result[:2])
	}
	//第二轮消费
	count = q.Dequeue(result)
	if count != 2 || result[0] != 3 || result[1] != 4 {
		t.Fatalf("关闭后消费数据不正确：%v", result[:2])
	}
	//第三轮消费
	count = q.Dequeue(result)
	if count != 0 {
		t.Fatalf("关闭后消费数据不正确：%v", result[:2])
	}
}

// TestLenAndCap 测试 Len 和 Cap 方法
func TestLenAndCap(t *testing.T) {
	q := New[int](16)

	if q.Cap() != 16 {
		t.Fatalf("容量期望 16，实际%d", q.Cap())
	}

	if q.Len() != 0 {
		t.Fatalf("空队列长度期望 0，实际%d", q.Len())
	}

	for i := 0; i < 5; i++ {
		q.Enqueue(i)
	}

	if q.Len() != 5 {
		t.Fatalf("长度期望 5，实际%d", q.Len())
	}
}

// TestStringType 测试使用字符串类型
func TestStringType(t *testing.T) {
	q := New[string](4)

	q.Enqueue("hello")
	q.Enqueue("world")

	result := make([]string, 10)
	count := q.Dequeue(result)

	if count != 2 || result[0] != "hello" || result[1] != "world" {
		t.Fatalf("字符串测试失败：%v", result[:2])
	}
}

// TestLargeCapacity 测试大容量队列
func TestLargeCapacity(t *testing.T) {
	q := New[int](10240) // 10K 容量

	// 填充大部分
	for i := 0; i < 8000; i++ {
		if !q.Enqueue(i) {
			t.Fatalf("入队%d失败", i)
		}
	}

	if q.Len() != 8000 {
		t.Fatalf("长度期望 8000，实际%d", q.Len())
	}

	// 消费一部分
	result := make([]int, 5000)
	count := q.Dequeue(result)
	if count != 5000 {
		t.Fatalf("期望消费 5000，实际%d", count)
	}

	if q.Len() != 3000 {
		t.Fatalf("消费后长度期望 3000，实际%d", q.Len())
	}
}

// TestCapacityAlignment 测试容量对齐到 2 的幂
func TestCapacityAlignment(t *testing.T) {
	testCases := []struct {
		input    int
		expected int
	}{
		{1, 2},
		{2, 2},
		{3, 4},
		{5, 8},
		{10, 16},
		{100, 128},
		{1024, 1024},
		{1025, 2048},
		{65535, 65536},
		{65536, 65536},
		{65537, 65536},
	}

	for _, tc := range testCases {
		q := New[int](tc.input)
		if q.Cap() != tc.expected {
			t.Fatalf("输入%d，期望容量%d，实际%d", tc.input, tc.expected, q.Cap())
		}
	}
}

// TestNegativeCapacity 测试负容量处理
func TestNegativeCapacity(t *testing.T) {
	q := New[int](-10)
	if q.Cap() != 2 {
		t.Fatalf("负容量应该默认为 2，实际%d", q.Cap())
	}

	// 应该能正常使用
	if !q.Enqueue(1) {
		t.Fatal("入队失败")
	}
}

// TestZeroCapacity 测试零容量处理
func TestZeroCapacity(t *testing.T) {
	q := New[int](0)
	if q.Cap() != 2 {
		t.Fatalf("零容量应该默认为 2，实际%d", q.Cap())
	}

	// 应该能正常使用
	if !q.Enqueue(1) {
		t.Fatal("入队失败")
	}
}

// TestSingleElement 测试单个元素的完整周期
func TestSingleElement(t *testing.T) {
	q := New[int](8)
	result := make([]int, 1)

	// 多次入队单个元素并立即出队
	for i := 0; i < 100; i++ {
		if !q.Enqueue(i) {
			t.Fatalf("第%d次入队失败", i)
		}
		count := q.Dequeue(result)
		if count != 1 || result[0] != i {
			t.Fatalf("第%d次出队失败：count=%d, value=%d", i, count, result[0])
		}
	}
}

// TestPartialBatchDequeue 测试部分批量出队
func TestPartialBatchDequeue(t *testing.T) {
	q := New[int](8)

	// 入队 3 个
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	// 尝试批量出队 5 个，应该只返回 3 个
	result := make([]int, 5)
	count := q.Dequeue(result)

	if count != 3 {
		t.Fatalf("期望出队 3 个，实际%d", count)
	}
	if result[0] != 1 || result[1] != 2 || result[2] != 3 {
		t.Fatalf("数据不正确：%v", result[:3])
	}
}

// TestManyRounds 测试多轮完整循环
func TestManyRounds(t *testing.T) {
	q := New[int](16)
	result := make([]int, 16)

	// 进行 100 轮完整循环
	for round := 0; round < 100; round++ {
		// 填满
		for i := 0; i < 16; i++ {
			if !q.Enqueue(round*100 + i) {
				t.Fatalf("第%d轮入队%d失败", round, i)
			}
		}

		// 全部出队
		count := q.Dequeue(result)
		if count != 16 {
			t.Fatalf("第%d轮期望出队 16 个，实际%d", round, count)
		}

		for i := 0; i < 16; i++ {
			if result[i] != round*100+i {
				t.Fatalf("第%d轮数据 [%d] 错误：期望%d，实际%d", round, i, round*100+i, result[i])
			}
		}
	}
}

// TestConcurrentFullWrite 测试并发写入直到满
func TestConcurrentFullWrite(t *testing.T) {
	q := New[int](100)
	wg := &sync.WaitGroup{}
	capacity := q.Cap() // 实际容量（会被对齐到 128）

	totalSuccess := 0
	var mu sync.Mutex

	// 10 个生产者，总共写 200 个（会超过容量）
	for p := 0; p < 10; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			success := 0
			for i := 0; i < 20; i++ {
				if q.Enqueue(producerID*100 + i) {
					success++
				}
			}
			mu.Lock()
			totalSuccess += success
			mu.Unlock()
		}(p)
	}
	wg.Wait()

	// 验证总成功次数不超过容量
	if totalSuccess > capacity {
		t.Fatalf("总成功次数%d超过容量%d", totalSuccess, capacity)
	}

	if q.Len() != totalSuccess {
		t.Fatalf("队列长度%d不等于总成功次数%d", q.Len(), totalSuccess)
	}
}

// TestProducerFasterThanConsumer 测试生产快于消费的场景
// 模拟：生产者快速写入，填满队列后验证队列满的行为
func TestProducerFasterThanConsumer(t *testing.T) {
	q := New[int](16) // 小容量，容易触发队列满的情况

	// 快速填满队列
	for i := 0; i < 16; i++ {
		if !q.Enqueue(i) {
			t.Fatalf("入队%d失败", i)
		}
	}

	// 验证队列已满
	if q.Enqueue(999) {
		t.Fatal("队列满时不应允许入队")
	}

	// 消费一半，模拟慢速消费
	result := make([]int, 8)
	count := q.Dequeue(result)
	if count != 8 {
		t.Fatalf("期望消费 8 个，实际%d", count)
	}

	// 现在应该可以再次入队
	if !q.Enqueue(100) {
		t.Fatal("消费后应允许入队")
	}

	// 关闭队列
	q.Close()

	// 消费剩余的元素
	for {
		count = q.Dequeue(result)
		if count == 0 {
			break
		}
	}

	if q.Len() != 0 {
		t.Fatalf("队列应为空，实际长度%d", q.Len())
	}
}

// TestConsumerFasterThanProducer 测试消费快于生产的场景
// 模拟：生产者写入少量数据，消费者频繁读取，验证队列为空时的行为
func TestConsumerFasterThanProducer(t *testing.T) {
	q := New[int](32)

	// 先入队少量数据
	for i := 0; i < 10; i++ {
		if !q.Enqueue(i) {
			t.Fatalf("入队%d失败", i)
		}
	}

	// 消费者快速读取多次，包括队列为空的情况
	result := make([]int, 5)
	count := q.Dequeue(result)
	if count != 5 || result[0] != 0 || result[4] != 4 {
		t.Fatalf("第一次消费数据不正确")
	}

	count = q.Dequeue(result)
	if count != 5 || result[0] != 5 || result[4] != 9 {
		t.Fatalf("第二次消费数据不正确")
	}

	// 队列为空，应返回 0
	count = q.Len()
	if count != 0 {
		t.Fatalf("队列空时应返回 0，实际%d", count)
	}

	// 再次入队数据
	for i := 20; i < 25; i++ {
		if !q.Enqueue(i) {
			t.Fatalf("入队%d失败", i)
		}
	}

	// 消费新数据
	count = q.Dequeue(result)
	if count != 5 || result[0] != 20 || result[4] != 24 {
		t.Fatalf("第三次消费数据不正确")
	}
}

// TestVariableProducerConsumerSpeeds 测试生产消费速度动态变化的场景
// 模拟：生产消费速度交替变化，验证队列在满和空状态间切换时仍能正常工作
func TestVariableProducerConsumerSpeeds(t *testing.T) {
	q := New[int](8) // 小容量，快速触发满/空状态

	// 多次交替填满和清空队列
	for round := 0; round < 20; round++ {
		// 填满队列
		for i := 0; i < 8; i++ {
			if !q.Enqueue(round*100 + i) {
				t.Fatalf("第%d轮入队%d失败", round, i)
			}
		}

		// 验证队列已满
		if q.Enqueue(999) {
			t.Fatalf("第%d轮队列满时不应允许入队", round)
		}

		// 消费全部
		result := make([]int, 8)
		count := q.Dequeue(result)
		if count != 8 {
			t.Fatalf("第%d轮期望消费 8 个，实际%d", round, count)
		}

		// 验证数据正确
		for i := 0; i < 8; i++ {
			if result[i] != round*100+i {
				t.Fatalf("第%d轮数据 [%d] 错误：期望%d，实际%d", round, i, round*100+i, result[i])
			}
		}

		// 验证队列已空
		count = q.Len()
		if count != 0 {
			t.Fatalf("第%d轮队列空时应返回 0，实际%d", round, count)
		}
	}
}

// TestBurstProducerConsumer 测试突发生产消费场景
// 模拟：生产者突然大量写入，消费者突然大量读取，验证队列在高负载下的稳定性
func TestBurstProducerConsumer(t *testing.T) {
	q := New[int](32) // 较小容量，容易触发满/空状态
	wg := &sync.WaitGroup{}
	total := 500

	// 共享切片（需要锁保护）
	var dequeued []int
	consumerDone := make(chan struct{})

	// 启动消费者（突发读取模式）
	go func() {
		defer func() { consumerDone <- struct{}{} }()
		burstCount := 0
		for {
			collected := len(dequeued)

			// 收集到500个元素时退出
			if collected >= total {
				return
			}

			// 突发批量读取：连续读取多次再暂停
			for j := 0; j < 3; j++ { // 一次突发读 3 批
				result := make([]int, 50)
				count := q.Dequeue(result)
				if count > 0 {
					dequeued = append(dequeued, result[:count]...)
				} else {
					return
				}
			}
			// 突发后暂停，模拟消费者慢速期
			burstCount++
			if burstCount%5 == 0 {
				runtime.Gosched() // 每 5 次突发让出 CPU，让生产者追赶
			}
		}
	}()

	// 5 个生产者：突发生产模式
	for p := 0; p < 5; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			burstCount := 0
			for i := 0; i < total/5; i++ {
				item := producerID*(total/5) + i
				for !q.Enqueue(item) {
					runtime.Gosched()
				}
				// 每 10 次快速写入一批后暂停
				if i > 0 && i%10 == 0 {
					burstCount++
					if burstCount%3 == 0 {
						// 每 3 次突发后稍长暂停，模拟生产者慢速期
						for k := 0; k < 5; k++ {
							runtime.Gosched()
						}
					} else {
						runtime.Gosched() // 普通暂停
					}
				}
			}
		}(p)
	}

	// 等待所有生产者完成
	wg.Wait()

	// 关闭队列
	q.Close()

	// 等待消费者完成
	<-consumerDone

	// 验证所有元素
	seen := make(map[int]bool)
	for _, item := range dequeued {
		if seen[item] {
			t.Fatalf("发现重复元素：%d", item)
		}
		seen[item] = true
	}

	if len(seen) != total {
		t.Fatalf("期望消费%d个唯一元素，实际%d", total, len(seen))
	}
}

// TestIsClosed 测试 IsClosedOnSafe 方法
func TestIsClosed(t *testing.T) {
	q := New[int](4)

	// 初始状态：未关闭
	if q.IsClosed() {
		t.Fatal("初始状态应该未关闭")
	}

	// 关闭队列
	q.Close()

	// 关闭后应返回 true
	if !q.IsClosed() {
		t.Fatal("关闭后应该返回 true")
	}
}

// TestDequeueEmptySlice 测试 Dequeue 空切片 panic
func TestDequeueEmptySlice(t *testing.T) {
	q := New[int](4)
	result := make([]int, 0)

	// 应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("期望 panic，但没有")
		}
	}()
	q.Dequeue(result)
}
