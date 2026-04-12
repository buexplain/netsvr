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

package slicePool

import (
	"sync"
	"testing"
)

// TestNewStrSlice 测试构造函数
func TestNewStrSlice(t *testing.T) {
	// 测试正常 step
	pool := NewStrSlice(16)
	if pool == nil {
		t.Fatal("NewStrSlice 返回 nil")
	}
	if pool.step != 16 {
		t.Errorf("step = %d, want 16", pool.step)
	}

	// 测试 step <= 0 的情况
	pool2 := NewStrSlice(0)
	if pool2.step != 16 {
		t.Errorf("step=0 时应该使用默认值 16，实际 %d", pool2.step)
	}

	pool3 := NewStrSlice(-1)
	if pool3.step != 16 {
		t.Errorf("step=-1 时应该使用默认值 16，实际 %d", pool3.step)
	}
}

// TestStrSliceGet 测试 Get 方法
func TestStrSliceGet(t *testing.T) {
	pool := NewStrSlice(16)

	// 测试 capacity > 0
	slice := pool.Get(10)
	if slice == nil {
		t.Fatal("Get(10) 返回 nil")
	}
	if cap(*slice) < 10 {
		t.Errorf("Get(10) 容量 = %d, 应该 >= 10", cap(*slice))
	}
	if len(*slice) != 0 {
		t.Errorf("Get(10) 长度 = %d, 应该为 0", len(*slice))
	}

	// 测试 capacity = 0（应该返回最小容量 1）
	slice2 := pool.Get(0)
	if slice2 == nil {
		t.Fatal("Get(0) 返回 nil")
	}
	if cap(*slice2) < 1 {
		t.Errorf("Get(0) 容量 = %d, 应该 >= 1", cap(*slice2))
	}

	// 测试 capacity < 0
	slice3 := pool.Get(-5)
	if slice3 == nil {
		t.Fatal("Get(-5) 返回 nil")
	}
	if cap(*slice3) < 1 {
		t.Errorf("Get(-5) 容量 = %d, 应该 >= 1", cap(*slice3))
	}

	// 测试大容量
	slice4 := pool.Get(100)
	if cap(*slice4) < 100 {
		t.Errorf("Get(100) 容量 = %d, 应该 >= 100", cap(*slice4))
	}
}

// TestStrSlicePut 测试 Put 方法
func TestStrSlicePut(t *testing.T) {
	pool := NewStrSlice(16)

	// 获取切片
	slice := pool.Get(10)

	// 添加一些数据
	for i := 0; i < 5; i++ {
		*slice = append(*slice, "test")
	}

	if len(*slice) != 5 {
		t.Errorf("添加后长度 = %d, 期望 5", len(*slice))
	}

	// 归还到池
	pool.Put(slice)

	// 验证长度被重置为 0
	if len(*slice) != 0 {
		t.Errorf("Put 后长度 = %d, 期望 0", len(*slice))
	}

	// 再次获取，验证复用
	slice2 := pool.Get(10)
	if slice2 == nil {
		t.Fatal("第二次 Get 返回 nil")
	}
	// 长度应该还是 0
	if len(*slice2) != 0 {
		t.Errorf("复用的切片长度 = %d, 期望 0", len(*slice2))
	}

	pool.Put(slice2)
}

// TestStrSlicePoolReuse 测试对象池复用
func TestStrSlicePoolReuse(t *testing.T) {
	pool := NewStrSlice(16)

	// 第一次获取
	slice1 := pool.Get(10)
	cap1 := cap(*slice1)
	pool.Put(slice1)

	// 第二次获取，应该从池中复用
	slice2 := pool.Get(10)
	cap2 := cap(*slice2)

	// 容量应该相同或相近
	if cap2 < cap1/2 {
		t.Logf("警告：池复用可能未生效，cap1=%d, cap2=%d", cap1, cap2)
	}

	pool.Put(slice2)
}

// TestStrSliceDifferentCapacityLevels 测试不同容量分级
func TestStrSliceDifferentCapacityLevels(t *testing.T) {
	pool := NewStrSlice(16)

	// 获取不同容量的切片
	slice1 := pool.Get(10) // poolIndex = 1, cap = 16
	slice2 := pool.Get(20) // poolIndex = 2, cap = 32
	slice3 := pool.Get(50) // poolIndex = 4, cap = 64

	if cap(*slice1) != 16 {
		t.Errorf("Get(10) 容量 = %d, 期望 16", cap(*slice1))
	}
	if cap(*slice2) != 32 {
		t.Errorf("Get(20) 容量 = %d, 期望 32", cap(*slice2))
	}
	if cap(*slice3) != 64 {
		t.Errorf("Get(50) 容量 = %d, 期望 64", cap(*slice3))
	}

	pool.Put(slice1)
	pool.Put(slice2)
	pool.Put(slice3)
}

// TestStrSliceConcurrentAccess 测试并发安全性
func TestStrSliceConcurrentAccess(t *testing.T) {
	pool := NewStrSlice(16)
	var wg sync.WaitGroup

	concurrentOps := 100

	// 并发 Get 和 Put
	for i := 0; i < concurrentOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			slice := pool.Get(10)
			// 模拟使用
			for j := 0; j < 5; j++ {
				*slice = append(*slice, "test")
			}
			pool.Put(slice)
		}(i)
	}

	wg.Wait()

	// 如果没有 panic，说明并发安全
}

// TestStrSliceMultiplePuts 测试多次 Put
func TestStrSliceMultiplePuts(t *testing.T) {
	pool := NewStrSlice(16)

	slice := pool.Get(10)

	// 多次 Put 同一个切片（虽然不应该这样做，但要确保不 panic）
	pool.Put(slice)
	pool.Put(slice)
	pool.Put(slice)

	// 如果能到这里，说明没有 panic
}

// TestStrSliceDataIntegrity 测试数据完整性
func TestStrSliceDataIntegrity(t *testing.T) {
	pool := NewStrSlice(16)

	// 第一次使用
	slice1 := pool.Get(10)
	testData := []string{"a", "b", "c"}
	for _, s := range testData {
		*slice1 = append(*slice1, s)
	}

	if len(*slice1) != 3 {
		t.Errorf("第一次使用长度 = %d, 期望 3", len(*slice1))
	}

	pool.Put(slice1)

	// 第二次获取，验证数据已被清空
	slice2 := pool.Get(10)
	if len(*slice2) != 0 {
		t.Errorf("复用后长度 = %d, 期望 0（数据应已清空）", len(*slice2))
	}
	if cap(*slice2) < 3 {
		t.Logf("警告：容量可能被重置，原容量 >= 3, 现容量 = %d", cap(*slice2))
	}

	pool.Put(slice2)
}

// BenchmarkStrSliceGet 性能基准测试
func BenchmarkStrSliceGet(b *testing.B) {
	pool := NewStrSlice(16)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice := pool.Get(10)
		pool.Put(slice)
	}
}

// BenchmarkStrSliceGetParallel 并发性能基准测试
func BenchmarkStrSliceGetParallel(b *testing.B) {
	pool := NewStrSlice(16)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			slice := pool.Get(10)
			pool.Put(slice)
		}
	})
}
