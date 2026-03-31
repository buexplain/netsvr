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
// 注意：sequence 和 data 紧挨着，因为它们总是一起访问
// 如果 T 很大，考虑在 sequence 和 data 之间添加填充
type slot[T any] struct {
	sequence atomic.Uint64 // 序列号，用于同步和判断槽位状态
	_        [cacheLineSize - 8]byte
	data     T // 实际存储的数据
}

// slotAccess 通过 base + stride 访问 slot 数组
type slotAccess[T any] struct {
	seqOffset  uintptr        // sequence 字段在结构体中的偏移量
	dataOffset uintptr        // data 字段在结构体中的偏移量
	stride     uintptr        // 相邻元素之间的内存间距（字节）
	base       unsafe.Pointer // 指向数组起始位置的指针
}

func (s *slotAccess[T]) sequenceAt(i uint64) *atomic.Uint64 {
	return (*atomic.Uint64)(unsafe.Add(s.base, s.stride*uintptr(i)+s.seqOffset))
}

func (s *slotAccess[T]) dataAt(i uint64) *T {
	return (*T)(unsafe.Add(s.base, s.stride*uintptr(i)+s.dataOffset))
}

// Queue 是一个无锁、有界、多生产者单消费者 (MPSC) 的环形队列。
//
// 内存布局优化：
//   - head: 生产者竞争写入，独立 cache line
//   - tail: 消费者写入，独立 cache line
//   - mask, closed, noticer: 控制字段，共享一个 cache line（低频访问）
//   - slots: 数据区域，通过 stride 确保每个逻辑 slot 独立
type Queue[T any] struct {
	// 生产者高频竞争区域（128 bytes）
	head atomic.Uint64
	_    [cacheLineSize - unsafe.Sizeof(atomic.Uint64{})]byte

	// 消费者高频写入区域（128 bytes）
	tail atomic.Uint64
	_    [cacheLineSize - unsafe.Sizeof(atomic.Uint64{})]byte

	// 控制区域（低频访问，共享 128 bytes）
	mask   uint64
	closed atomic.Bool
	_      [cacheLineSize - unsafe.Sizeof(uint64(0)) - unsafe.Sizeof(atomic.Bool{})]byte

	// 通知通道（指针，8 bytes）
	noticer chan struct{}

	// 数据访问器（32 bytes on 64-bit）
	slots slotAccess[T]

	// GC 引用（保持内存不被回收）
	reference []slot[T]

	// 零值（用于重置）
	zero T
}

// NewQueue 创建一个队列（padded 布局，消除伪共享）。
// capacity 会自动向上取整到最近的 2 的幂。
// 通过 over-allocate 使每个逻辑 slot 占满整条 cache line。
func NewQueue[T any](capacity int) *Queue[T] {
	if capacity <= 1 {
		capacity = 2
	} else if capacity > 65536 {
		capacity = 65536
	}

	// 对齐到 2 的幂
	capacity--
	capacity |= capacity >> 1
	capacity |= capacity >> 2
	capacity |= capacity >> 4
	capacity |= capacity >> 8
	capacity |= capacity >> 16
	capacity++

	// 计算 slot 大小和 stride
	slotSize := unsafe.Sizeof(slot[T]{})

	// 确保每个逻辑 slot 至少占满一个 cache line
	// 如果 slot 本身大于 cacheLineSize，则使用 slotSize
	stride := slotSize
	if stride < cacheLineSize {
		stride = cacheLineSize
	}

	// 计算每个逻辑 slot 需要多少个物理 slot
	slotsPerLogical := uintptr(1)
	if slotSize > 0 {
		slotsPerLogical = (stride + slotSize - 1) / slotSize
		if slotsPerLogical*slotSize < cacheLineSize {
			slotsPerLogical++
			stride = slotsPerLogical * slotSize
		}
	}

	buf := make([]slot[T], uint64(capacity)*uint64(slotsPerLogical))
	var zero T
	q := &Queue[T]{
		mask:      uint64(capacity - 1),
		noticer:   make(chan struct{}, 1),
		reference: buf,
		zero:      zero,
	}
	q.slots = newSlotAccess(buf, stride)
	for i := uint64(0); i < uint64(capacity); i++ {
		q.slots.sequenceAt(i).Store(i)
	}
	return q
}

func newSlotAccess[T any](buf []slot[T], stride uintptr) slotAccess[T] {
	return slotAccess[T]{
		base:       unsafe.Pointer(&buf[0]),
		seqOffset:  unsafe.Offsetof(buf[0].sequence),
		dataOffset: unsafe.Offsetof(buf[0].data),
		stride:     stride,
	}
}

// Enqueue 非阻塞入队单个 T。
// 多生产者竞争时自动重试 CAS，队列满或已关闭时返回 false。
func (q *Queue[T]) Enqueue(item T) bool {
	for { // 自旋循环，持续尝试直到成功或失败
		if q.closed.Load() { // 检查队列是否已关闭
			return false // 队列已关闭，拒绝新元素入队，返回失败
		}

		head := q.head.Load()                   // 加载当前队头位置（原子读取）
		index := head & q.mask                  // 通过位运算计算环形索引（相当于 head % capacity）
		seq := q.slots.sequenceAt(index).Load() // 加载该位置的序列号（原子读取）

		if seq == head { // 序列号等于队头，说明此槽位可写（消费者已消费）
			if q.head.CompareAndSwap(head, head+1) { // CAS 尝试占用此位置（原子操作）
				*q.slots.dataAt(index) = item            // 写入数据到槽位
				q.slots.sequenceAt(index).Store(seq + 1) // 序列号加 1，表示数据已就绪，消费者可读取

				// 通知消费者有新数据（非阻塞发送）
				select {
				case q.noticer <- emptyStruct: // 尝试发送通知信号
					return true // 发送成功，入队完成
				default: // 通道已满或有其他生产者抢先发送
					return true // 无需重复发送，入队同样成功
				}
			}
			runtime.Gosched() // CAS 失败（被其他生产者抢先），让出 CPU 时间片，减少空转
			continue          // 重新尝试下一轮入队
		}

		if seq < head { // 序列号小于队头，说明生产者跑到了消费者前面（队列已满）
			if q.head.Load() == head { // 二次确认队头未变化（避免假阳性）
				return false // 队头确实未变，队列已满，返回失败
			}
			continue // 队头已变化，说明是假阳性，重新尝试
		}
		// seq > head：序列号大于队头，说明消费者跑到了生产者前面（队列已空）
		// 理论上不会出现，continue 重试即可
	}
}

// TryEnqueue 尝试入队单个 T。入队列成功返回 true，失败返回 false。失败的原因有：CAS 失败（多生产者竞争）或队列满
func (q *Queue[T]) TryEnqueue(item T) bool {
	if q.closed.Load() { // 检查队列是否已关闭（原子操作）
		return false // 队列已关闭，拒绝新元素入队，返回失败
	}

	head := q.head.Load()                   // 加载当前队头位置（原子读取）
	index := head & q.mask                  // 通过位运算计算环形索引（相当于 head % capacity，利用 mask 快速取模）
	seq := q.slots.sequenceAt(index).Load() // 加载目标索引位置的序列号（原子读取）

	if seq == head { // 序列号等于队头，说明此槽位空闲可写（消费者已完成消费）
		if q.head.CompareAndSwap(head, head+1) { // CAS 尝试占用此位置（原子比较并交换）
			*q.slots.dataAt(index) = item            // 将数据写入槽位的 data 字段
			q.slots.sequenceAt(index).Store(seq + 1) // 序列号加 1，标记数据已就绪，通知消费者可读取

			// 通知消费者有新数据（非阻塞发送）
			select {
			case q.noticer <- emptyStruct: // 尝试向通知通道发送信号
				return true // 发送成功，入队完成
			default: // 通道已满或其他生产者已发送通知
				return true // 无需重复发送，入队同样成功
			}
		}
	}
	return false // CAS 失败（多生产者竞争）或队列满（seq != head），立即返回失败，不重试
}

// Dequeue 批量出队 T。返回值表示实际读取的元素数量，返回0的原因是：队列已关闭且无数据
func (q *Queue[T]) Dequeue(result []T) int {
	limit := len(result) // 获取结果切片的长度，决定单次批量出队的最大数量
	if limit == 0 {      // 检查结果切片是否为空
		panic("result slice is empty") // 结果切片为空时 panic，避免无意义的调用
	}

loop: // 标签，用于在特定条件下重新开始整个读取流程
	tail := q.tail.Load() // 加载当前队尾位置（原子读取，消费者独占修改）
	count := 0            // 已读取元素数量计数器，初始化为 0

	for count < limit { // 循环读取数据，直到达到结果切片容量上限
		index := tail & q.mask                  // 通过位运算计算环形索引（相当于 tail % capacity）
		seq := q.slots.sequenceAt(index).Load() // 加载该位置的序列号（原子读取）

		if seq != tail+1 { // 序列号不等于 tail+1，说明此位置无就绪数据
			break // 跳出 for 循环，进入等待或返回逻辑
		}

		result[count] = *q.slots.dataAt(index) // 读取槽位数据到结果切片
		*q.slots.dataAt(index) = q.zero        // 清除原数据（帮助 GC 回收，避免内存泄漏）
		tail++                                 // 队尾指针前移，指向下一个待消费位置
		count++                                // 已读取计数加 1
	}

	if count == 0 { // 没有读取到任何数据（队列为空或暂时未就绪）
		if q.closed.Load() == false { // 队列未关闭
			<-q.noticer // 阻塞等待通知信号（可能是新数据写入，也可能是 close 操作）
		}
		if q.closed.Load() == true { // 队列已关闭（等待期间可能被关闭）
			if q.Len() > 0 { // 检查队列是否还有剩余数据（双重检查）
				goto loop // 队列非空，跳转到 loop 标签处重新读取
			}
			return 0 // 队列已关闭且无数据，返回 0 表示结束
		}
		goto loop // 等待到了新数据，跳转到 loop 标签处重新读取
	}

	q.tail.Store(tail) // 批量更新队尾指针（原子存储，一次性提交所有消费）

	baseTail := tail - uint64(count)             // 计算起始队尾位置（本次消费前的 tail 值）
	for i := uint64(0); i < uint64(count); i++ { // 遍历所有已消费的槽位，逐个释放
		releaseTail := baseTail + i   // 计算该槽位被消费时的总次数（绝对位置）
		index := releaseTail & q.mask // 计算环形索引
		// 修复：释放槽位为 (releaseTail + mask + 1)
		// 这样下一轮入队时，生产者看到的 sequence[index] = releaseTail + mask + 1
		// 而期望值是 releaseTail + 1，满足 sequence[index] == head 条件
		q.slots.sequenceAt(index).Store(releaseTail + q.mask + 1) // 释放槽位，重置序列号供下一轮复用
	}

	return count // 返回实际读取的元素数量
}

func (q *Queue[T]) Len() int {
	t := q.tail.Load()
	h := q.head.Load()
	if h < t {
		return 0
	}
	return int(h - t)
}

func (q *Queue[T]) Cap() int {
	return int(q.mask) + 1
}

func (q *Queue[T]) IsClosed() bool {
	return q.closed.Load()
}

func (q *Queue[T]) Close() {
	if q.closed.CompareAndSwap(false, true) {
		select {
		case q.noticer <- emptyStruct:
		default:
		}
	}
}
