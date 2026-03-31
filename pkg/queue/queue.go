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
	"sync/atomic"
	"unsafe"
)

// cacheLineSize 用于防止伪共享 (False Sharing)。
// 现代 CPU 通常为 64 或 128 字节。
const cacheLineSize = 128

var emptyStruct = struct{}{}

// slot 代表环形缓冲区中的一个槽位
type slot[T any] struct {
	sequence atomic.Uint64 // 序列号，用于同步和判断槽位状态
	data     T             // 实际存储的数据
}

// slotAccess 通过 base + stride 访问 slot 数组
type slotAccess[T any] struct {
	seqOffset  uintptr        // sequence 字段在结构体中的偏移量
	dataOffset uintptr        // data 字段在结构体中的偏移量
	stride     uintptr        // 相邻元素之间的内存间距（字节）
	base       unsafe.Pointer // 指向数组起始位置的指针
}

// sequenceAt 返回索引 i 处的 sequence 字段的指针
func (s *slotAccess[T]) sequenceAt(i uint64) *atomic.Uint64 {
	return (*atomic.Uint64)(unsafe.Add(s.base, s.stride*uintptr(i)+s.seqOffset))
}

// dataAt 返回索引 i 处的 data 字段的指针
func (s *slotAccess[T]) dataAt(i uint64) *T {
	return (*T)(unsafe.Add(s.base, s.stride*uintptr(i)+s.dataOffset))
}

// Queue 是一个无锁、有界、多生产者单消费者 (MPSC) 的环形队列。
// 专门优化用于存储 [2][]byte 类型的数据对。
//
// 特性：
//   - 无锁设计：多生产者通过 CAS 竞争，消费者无竞争
//   - 批量出队：支持高效批量出队操作
//   - Cache line padding: 防止伪共享
//   - CAS 退避：减少竞争开销
//
// 使用约束：
//   - Enqueue: 多线程安全
//   - Dequeue: 仅限单消费者调用
type Queue[T any] struct {
	head atomic.Uint64       // 队头指针，多个生产者 CAS 竞争
	_    [cacheLineSize]byte // 填充，防止 head 和 tail 在同一 cache line

	tail atomic.Uint64       // 队尾指针，仅消费者修改
	_    [cacheLineSize]byte // 填充，防止 tail 和 mask 在同一 cache line

	mask      uint64        // 容量减 1，用于快速计算索引（位运算）
	closed    atomic.Bool   // 队列关闭标志
	slots     slotAccess[T] // 槽位访问器，用于快速定位元素
	noticer   chan struct{} // 用于通知消费者
	reference []slot[T]     // 实际缓冲区，保持 GC 引用
	zero      T             // 用于初始化零值
}

// NewQueue 创建一个队列（padded 布局，消除伪共享）。
// capacity 会自动向上取整到最近的 2 的幂。
// 通过 over-allocate 使每个逻辑 slot 占满整条 cache line。
// 内存开销：capacity × max(cacheLineSize, sizeof(slot))。
func NewQueue[T any](capacity int) *Queue[T] {
	if capacity <= 1 { // 处理非法容量值和容量为1时特殊处理
		capacity = 2 // 默认最小容量为 2
	} else if capacity > 65536 {
		capacity = 65536 // 避免超过 65536
	}

	capacity--                 // 减 1，为后续对齐做准备
	capacity |= capacity >> 1  // 位运算：将容量对齐到 2 的幂
	capacity |= capacity >> 2  // 传播最高位的 1 到右侧
	capacity |= capacity >> 4  // 继续传播
	capacity |= capacity >> 8  // 继续传播
	capacity |= capacity >> 16 // 继续传播
	capacity += 1              // 加 1，得到最终的 2 的幂容量

	slotSize := unsafe.Sizeof(slot[T]{})                                     // 计算单个槽位的大小
	stride := (slotSize + cacheLineSize - 1) / cacheLineSize * cacheLineSize // 向上取整到 cache line 倍数
	slotsPerLogical := stride / slotSize                                     // 计算每个逻辑槽位需要多少个物理槽位
	if stride%slotSize != 0 {                                                // 如果不能整除
		slotsPerLogical++                   // 增加一个槽位
		stride = slotsPerLogical * slotSize // 重新计算实际步长
	}

	buf := make([]slot[T], uint64(capacity)*uint64(slotsPerLogical)) // 分配实际缓冲区
	var zero T
	q := &Queue[T]{
		mask:      uint64(capacity - 1),   // 设置 mask 为容量减 1
		noticer:   make(chan struct{}, 1), // 创建通知通道，用于通知消费者，必须是有缓冲的
		reference: buf,                    // 保存缓冲区引用
		zero:      zero,
	}
	q.slots = newSlotAccess(buf, stride)            // 初始化槽位访问器
	for i := uint64(0); i < uint64(capacity); i++ { // 初始化每个槽位的序列号
		q.slots.sequenceAt(i).Store(i) // 初始序列号等于索引
	}
	return q
}

// newSlotAccess 创建一个新的 slotAccess 结构体，用于高效访问字节槽数组。
// 通过存储基地址、字段偏移量和步长，避免重复计算内存位置。
// 参数:
//   - buf: slot 类型的切片，作为访问的数据源
//   - stride: uintptr 类型，表示相邻元素之间的内存间距
//
// 返回:
//   - slotAccess: 包含访问信息的结构体，可用于快速定位数组中任意元素的字段
func newSlotAccess[T any](buf []slot[T], stride uintptr) slotAccess[T] {
	return slotAccess[T]{
		base:       unsafe.Pointer(&buf[0]),          // 获取数组首地址
		seqOffset:  unsafe.Offsetof(buf[0].sequence), // 计算 sequence 字段偏移量
		dataOffset: unsafe.Offsetof(buf[0].data),     // 计算 data 字段偏移量
		stride:     stride,                           // 设置步长
	}
}

// Enqueue 非阻塞入队单个 T。
// 多生产者竞争时自动重试 CAS，队列满或已关闭时返回 false。
func (q *Queue[T]) Enqueue(item T) bool {
	for { // 自旋循环，直到成功或失败
		if q.closed.Load() { // 检查队列是否已关闭
			return false // 已关闭，返回失败
		}

		head := q.head.Load()                   // 加载当前队头位置
		index := head & q.mask                  // 通过位运算计算环形索引
		seq := q.slots.sequenceAt(index).Load() // 加载该位置的序列号

		if seq == head { // 序列号等于队头，说明此位置可写
			if q.head.CompareAndSwap(head, head+1) { // CAS 尝试占用此位置
				*q.slots.dataAt(index) = item            // 写入数据
				q.slots.sequenceAt(index).Store(seq + 1) // 在当前序列号基础上加1，表示数据已就绪
				select {
				case q.noticer <- emptyStruct: // 通知消费者
					return true // 入队成功
				default:
					return true // 入队成功
				}
			}
			runtime.Gosched() // CAS 失败，让出 CPU 时间片
			continue          // 重试
		}

		if seq < head { // 序列号小于队头，说明队列已满（生产者跑到了消费者前面）
			if q.head.Load() == head { // 再次确认队头未变化
				return false // 队头未变，确实满了，返回 false
			}
			continue // 队头已变，假阳性，continue 重试
		}
		// 序列号大于队头，说明队列已空（消费者跑到了生产者前面）
	}
}

// TryEnqueue 非阻塞入队（单次尝试）。
// 队列满或 CAS 竞争失败时立即返回 false，不重试。
func (q *Queue[T]) TryEnqueue(item T) bool {
	if q.closed.Load() { // 检查队列是否已关闭
		return false // 已关闭，返回失败
	}

	head := q.head.Load()                   // 加载当前队头位置
	index := head & q.mask                  // 计算环形索引
	seq := q.slots.sequenceAt(index).Load() // 加载序列号

	if seq == head { // 可以写入
		if q.head.CompareAndSwap(head, head+1) { // 单次 CAS 尝试
			*q.slots.dataAt(index) = item            // 写入数据
			q.slots.sequenceAt(index).Store(seq + 1) // 在当前序列号基础上加1
			select {
			case q.noticer <- emptyStruct: // 通知消费者
				return true // 成功
			default:
				return true // 成功
			}
		}
	}
	return false // CAS 失败或队列满，直接返回
}

// Dequeue 批量出队（阻塞，两阶段提交）。
// result 切片长度决定最大批量，返回实际读取的数量；返回0表示队列已经关闭，且队列中没有数据。
func (q *Queue[T]) Dequeue(result []T) int {
	limit := len(result) // 最大可读取数量
	if limit == 0 {      // 结果数组为空
		panic("result slice is empty")
	}

loop:

	tail := q.tail.Load() // 加载当前队尾位置
	count := 0 // 已读取数量计数器

	// 读取数据，不释放槽位
	for count < limit { // 未达到上限前继续读取
		index := tail & q.mask                  // 计算环形索引
		seq := q.slots.sequenceAt(index).Load() // 加载序列号

		if seq != tail+1 { // 序列号不等于 tail+1，说明无数据
			break // 跳出循环
		}

		result[count] = *q.slots.dataAt(index) // 读取数据到结果数组
		*q.slots.dataAt(index) = q.zero        // 清除原数据（帮助 GC）
		tail++                                 // 队尾前移
		count++                                // 计数增加
	}

	if count == 0 { // 没有读取到任何数据
		if q.closed.Load() == false { // 队列未关闭
			<-q.noticer // 等待信号，也许是 生产者写入，也许是 close 退出
		}
		if q.closed.Load() == true { // 队列已关闭
			if q.Len() > 0 { // 队列非空
				goto loop // 重新开始读取
			}
			return 0 // 队列已关闭且已无数据，返回
		}
		goto loop // 等待到了新的数据，重新开始读取
	}

	// 先推进 tail，再释放所有槽位
	q.tail.Store(tail) // 更新队尾指针

	baseTail := tail - uint64(count)             // 计算起始队尾位置
	for i := uint64(0); i < uint64(count); i++ { // 遍历所有已消费的槽位
		releaseTail := baseTail + i   // 计算实际位置（该槽位被消费时的总次数）
		index := releaseTail & q.mask // 计算环形索引
		// 修复：释放槽位为 (releaseTail + mask + 1)
		// 这样下一轮入队时，生产者看到的 sequence[index] = releaseTail + mask + 1
		// 而期望值是 releaseTail + 1，满足 sequence[index] == head 条件
		q.slots.sequenceAt(index).Store(releaseTail + q.mask + 1) // 释放槽位，供下一轮复用
	}

	return count // 返回实际读取数量
}

// Len 返回队列中当前的估计元素数量。
func (q *Queue[T]) Len() int {
	t := q.tail.Load() // 加载队尾位置
	h := q.head.Load() // 加载队头位置

	if h < t { // 队头小于队尾（理论上不应该）
		return 0 // 返回 0
	}
	return int(h - t) // 直接返回差值作为队列长度
}

// Cap 返回队列总容量。
func (q *Queue[T]) Cap() int {
	return int(q.mask) + 1 // mask+1 即为容量
}

// IsClosed 检查队列是否已关闭。
func (q *Queue[T]) IsClosed() bool {
	return q.closed.Load() // 返回关闭标志
}

// Close 关闭队列。关闭后 Enqueue 返回 false，已入队数据仍可 Dequeue。
func (q *Queue[T]) Close() {
	if q.closed.CompareAndSwap(false, true) { // 设置关闭标志
		select {
		case q.noticer <- emptyStruct: // 通知消费者退出
		default:
		}
	}
}
