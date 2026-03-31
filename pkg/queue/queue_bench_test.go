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
	"testing"
	"time"
)

// BenchmarkEnqueue 入队性能基准测试
// 测试单线程入队的吞吐量
func BenchmarkEnqueue(b *testing.B) {
	q := NewQueue[int](10240)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
	}
}

// BenchmarkDequeue 批量出队性能基准测试
// 测试批量出队的吞吐量
func BenchmarkDequeue(b *testing.B) {
	q := NewQueue[int](10240)
	result := make([]int, 100)

	// 预填充
	for i := 0; i < b.N+100; i++ {
		q.Enqueue(i)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		q.Close()
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Dequeue(result)
	}
}

func concurrentEnqueueDequeue(b *testing.B, batchSize int) {
	q := NewQueue[int](1024)
	wg := &sync.WaitGroup{}
	numProducers := 4
	itemsPerProducer := b.N / numProducers
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		result := make([]int, batchSize)
		for {
			count := q.Dequeue(result)
			if count == 0 {
				return
			}
		}
	}()

	b.ResetTimer()
	for p := 0; p < numProducers; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			for i := 0; i < itemsPerProducer; i++ {
				for !q.Enqueue(producerID*100000 + i) {
					runtime.Gosched()
				}
			}
		}(p)
	}
	wg.Wait()
	q.Close()
	// 等待消费者完成
	<-consumerDone
}

// BenchmarkConcurrentEnqueueDequeue1 多生产者并发入队性能基准测试
// 测试多生产者并发场景下的吞吐量
func BenchmarkConcurrentEnqueueDequeue1(b *testing.B) {
	concurrentEnqueueDequeue(b, 1)
}

// BenchmarkConcurrentEnqueueDequeue8 多生产者并发入队性能基准测试
// 测试多生产者并发场景下的吞吐量
func BenchmarkConcurrentEnqueueDequeue8(b *testing.B) {
	concurrentEnqueueDequeue(b, 8)
}

// BenchmarkConcurrentEnqueueWithChannel 多生产者并发入队性能基准测试
// 测试多生产者并发场景下的吞吐量
func BenchmarkConcurrentEnqueueWithChannel(b *testing.B) {
	q := make(chan int, 1024)
	wg := &sync.WaitGroup{}
	numProducers := 4
	itemsPerProducer := b.N / numProducers
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		for {
			if _, ok := <-q; !ok {
				return
			}
		}
	}()

	b.ResetTimer()
	for p := 0; p < numProducers; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			for i := 0; i < itemsPerProducer; i++ {
				q <- producerID*100000 + i
			}
		}(p)
	}
	wg.Wait()
	close(q)
	<-consumerDone
}

// BenchmarkFullCycle 完整循环性能基准测试
// 测试队列在填满和清空循环下的性能
func BenchmarkFullCycle(b *testing.B) {
	q := NewQueue[int](64)
	result := make([]int, 64)

	b.ResetTimer()

	for round := 0; round < b.N; round++ {
		// 填满队列
		for i := 0; i < 64; i++ {
			q.Enqueue(round*100 + i)
		}
		// 清空队列
		q.Dequeue(result)
	}
}

// BenchmarkFullCycleWithChannel 完整循环性能基准测试
// 测试队列在填满和清空循环下的性能
func BenchmarkFullCycleWithChannel(b *testing.B) {
	q := make(chan int, 64)

	b.ResetTimer()

	for round := 0; round < b.N; round++ {
		// 填满队列
		for i := 0; i < 64; i++ {
			q <- round*100 + i
		}
		// 清空队列
		for i := 0; i < 64; i++ {
			<-q
		}
	}
}

// BenchmarkTryEnqueue TryEnqueue 单次尝试性能基准测试
// 测试非阻塞入队的吞吐量
func BenchmarkTryEnqueue(b *testing.B) {
	q := NewQueue[int](1024)
	// 预填充一半，确保不会频繁满
	for i := 0; i < 512; i++ {
		q.Enqueue(i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.TryEnqueue(i)
	}
}

// BenchmarkLen Len 方法性能基准测试
// 测试获取队列长度的性能
func BenchmarkLen(b *testing.B) {
	q := NewQueue[int](1024)
	for i := 0; i < 512; i++ {
		q.Enqueue(i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Len()
	}
}

// BenchmarkCap Cap 方法性能基准测试
// 测试获取队列容量的性能
func BenchmarkCap(b *testing.B) {
	q := NewQueue[int](1024)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Cap()
	}
}
