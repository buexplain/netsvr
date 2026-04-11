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

package binder

import (
	"sync"
	"testing"
)

// newTestCollect 创建一个新的 collect 实例用于测试
func newTestCollect() *collect {
	c := &collect{}
	for i := range c.shards {
		c.shards[i].data = make(map[string][]string, 128)
	}
	return c
}

// TestHashCustomerId 测试哈希函数的一致性
func TestHashCustomerId(t *testing.T) {
	tests := []struct {
		name       string
		customerId string
	}{
		{"empty", ""},
		{"simple", "customer1"},
		{"chinese", "客户测试"},
		{"special", "customer-with-special_chars!@#"},
		{"long", "this-is-a-very-long-customer-id-for-testing-purposes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashCustomerId(tt.customerId)
			hash2 := hashCustomerId(tt.customerId)
			// 同一输入应该产生相同输出
			if hash1 != hash2 {
				t.Errorf("hashCustomerId(%q) = %d, then %d, should be equal", tt.customerId, hash1, hash2)
			}
			// 结果应该在有效范围内
			if hash1 < 0 || hash1 >= shardCount {
				t.Errorf("hashCustomerId(%q) = %d, should be in [0, %d)", tt.customerId, hash1, shardCount)
			}
		})
	}
}

// TestLen 测试客户ID数量统计
func TestLen(t *testing.T) {
	tc := newTestCollect()

	// 初始状态应为 0
	if l := tc.Len(); l != 0 {
		t.Errorf("initial Len() = %d, want 0", l)
	}

	// 添加一些绑定关系
	tc.Set("customer1", "uniq1")
	tc.Set("customer2", "uniq2")
	tc.Set("customer3", "uniq3")
	if l := tc.Len(); l != 3 {
		t.Errorf("after adding 3 customers, Len() = %d, want 3", l)
	}

	// 同一客户添加多个 uniqId，数量不应变化
	tc.Set("customer1", "uniq4")
	if l := tc.Len(); l != 3 {
		t.Errorf("after adding another uniqId to customer1, Len() = %d, want 3", l)
	}

	// 删除一个 uniqId，客户仍存在
	tc.Del("customer1", "uniq1")
	if l := tc.Len(); l != 3 {
		t.Errorf("after deleting one uniqId from customer1, Len() = %d, want 3", l)
	}

	// 删除最后一个 uniqId，客户应被清理
	tc.Del("customer1", "uniq4")
	if l := tc.Len(); l != 2 {
		t.Errorf("after deleting last uniqId from customer1, Len() = %d, want 2", l)
	}
}

// TestSet 测试设置绑定关系
func TestSet(t *testing.T) {
	tc := newTestCollect()

	// 正常添加
	tc.Set("customer1", "uniq1")
	uniqIds := tc.GetUniqIdsByCustomerId("customer1")
	if len(uniqIds) != 1 || uniqIds[0] != "uniq1" {
		t.Errorf("after Set, got %v, want [uniq1]", uniqIds)
	}

	// 同一客户添加第二个 uniqId
	tc.Set("customer1", "uniq2")
	uniqIds = tc.GetUniqIdsByCustomerId("customer1")
	if len(uniqIds) != 2 {
		t.Errorf("customer1 should have 2 uniqIds, got %d", len(uniqIds))
	}

	// 验证两个 uniqId 都存在
	uniqIdMap := make(map[string]bool)
	for _, id := range uniqIds {
		uniqIdMap[id] = true
	}
	if !uniqIdMap["uniq1"] || !uniqIdMap["uniq2"] {
		t.Errorf("customer1 uniqIds = %v, missing some ids", uniqIds)
	}

	// 重复添加同一 uniqId（幂等性）
	tc.Set("customer1", "uniq1")
	uniqIds = tc.GetUniqIdsByCustomerId("customer1")
	if len(uniqIds) != 2 {
		t.Errorf("after duplicate Set, customer1 should still have 2 uniqIds, got %d", len(uniqIds))
	}

	// 不同客户可以绑定相同的 uniqId
	tc.Set("customer2", "uniq1")
	uniqIds2 := tc.GetUniqIdsByCustomerId("customer2")
	if len(uniqIds2) != 1 || uniqIds2[0] != "uniq1" {
		t.Errorf("customer2 should have uniq1, got %v", uniqIds2)
	}

	// 空字符串处理
	tc.Set("", "uniq3")
	tc.Set("customer3", "")
	// 空 customerId 仍然会被存储（根据实现）
	if l := tc.Len(); l < 2 {
		t.Errorf("after Set with empty strings, Len() = %d, want at least 2", l)
	}
}

// TestDel 测试删除绑定关系
func TestDel(t *testing.T) {
	tc := newTestCollect()

	// 准备数据
	tc.Set("customer1", "uniq1")
	tc.Set("customer1", "uniq2")
	tc.Set("customer1", "uniq3")
	tc.Set("customer2", "uniq4")

	// 删除不存在的客户
	tc.Del("nonexistent", "uniq1") // 不应该 panic
	if l := tc.Len(); l != 2 {
		t.Errorf("after Del nonexistent customer, Len() = %d, want 2", l)
	}

	// 删除不存在的 uniqId
	tc.Del("customer1", "nonexistent") // 不应该 panic
	uniqIds := tc.GetUniqIdsByCustomerId("customer1")
	if len(uniqIds) != 3 {
		t.Errorf("after Del nonexistent uniqId, customer1 should still have 3 uniqIds, got %d", len(uniqIds))
	}

	// 删除中间的 uniqId
	tc.Del("customer1", "uniq2")
	uniqIds = tc.GetUniqIdsByCustomerId("customer1")
	if len(uniqIds) != 2 {
		t.Errorf("after deleting uniq2, customer1 should have 2 uniqIds, got %d", len(uniqIds))
	}
	uniqIdMap := make(map[string]bool)
	for _, id := range uniqIds {
		uniqIdMap[id] = true
	}
	if !uniqIdMap["uniq1"] || !uniqIdMap["uniq3"] {
		t.Errorf("customer1 uniqIds = %v, want [uniq1, uniq3]", uniqIds)
	}

	// 删除最后一个 uniqId，客户应被清理
	tc.Del("customer1", "uniq1")
	tc.Del("customer1", "uniq3")
	if l := tc.Len(); l != 1 {
		t.Errorf("after deleting all uniqIds from customer1, Len() = %d, want 1", l)
	}
	uniqIds = tc.GetUniqIdsByCustomerId("customer1")
	if uniqIds != nil {
		t.Errorf("after deleting all uniqIds, GetUniqIdsByCustomerId should return nil, got %v", uniqIds)
	}
}

// TestGetCustomerIds 测试获取所有客户ID
func TestGetCustomerIds(t *testing.T) {
	tc := newTestCollect()

	// 空状态
	customerIds := tc.GetCustomerIds()
	if customerIds != nil {
		t.Errorf("GetCustomerIds() on empty = %v, want nil", customerIds)
	}

	// 添加数据
	tc.Set("customer1", "uniq1")
	tc.Set("customer2", "uniq2")
	tc.Set("customer3", "uniq3")

	customerIds = tc.GetCustomerIds()
	if len(customerIds) != 3 {
		t.Errorf("GetCustomerIds() returned %d customers, want 3", len(customerIds))
	}

	// 验证包含所有客户
	customerIdMap := make(map[string]bool)
	for _, id := range customerIds {
		customerIdMap[id] = true
	}
	if !customerIdMap["customer1"] || !customerIdMap["customer2"] || !customerIdMap["customer3"] {
		t.Errorf("GetCustomerIds() = %v, missing some customers", customerIds)
	}
}

// TestGetUniqIdsByCustomerId 测试根据客户ID获取uniqId列表
func TestGetUniqIdsByCustomerId(t *testing.T) {
	tc := newTestCollect()

	// 空字符串
	uniqIds := tc.GetUniqIdsByCustomerId("")
	if uniqIds != nil {
		t.Errorf("GetUniqIdsByCustomerId(\"\") = %v, want nil", uniqIds)
	}

	// 不存在的客户
	uniqIds = tc.GetUniqIdsByCustomerId("nonexistent")
	if uniqIds != nil {
		t.Errorf("GetUniqIdsByCustomerId(nonexistent) = %v, want nil", uniqIds)
	}

	// 添加数据
	tc.Set("customer1", "uniq1")
	tc.Set("customer1", "uniq2")
	tc.Set("customer1", "uniq3")

	// 获取 uniqId 列表
	uniqIds = tc.GetUniqIdsByCustomerId("customer1")
	if uniqIds == nil {
		t.Fatal("GetUniqIdsByCustomerId(customer1) returned nil")
	}
	if len(uniqIds) != 3 {
		t.Errorf("GetUniqIdsByCustomerId(customer1) returned %d uniqIds, want 3", len(uniqIds))
	}

	// 验证包含所有 uniqId
	uniqIdMap := make(map[string]bool)
	for _, id := range uniqIds {
		uniqIdMap[id] = true
	}
	if !uniqIdMap["uniq1"] || !uniqIdMap["uniq2"] || !uniqIdMap["uniq3"] {
		t.Errorf("GetUniqIdsByCustomerId(customer1) = %v, missing some uniqIds", uniqIds)
	}

	// 验证返回的是副本，修改不影响内部数据
	uniqIds[0] = "modified"
	uniqIds2 := tc.GetUniqIdsByCustomerId("customer1")
	if uniqIds2[0] == "modified" {
		t.Error("modifying returned slice should not affect internal data")
	}
}

// TestGetUniqIdsByCustomerIds 测试批量获取多个客户的uniqId列表
func TestGetUniqIdsByCustomerIds(t *testing.T) {
	tc := newTestCollect()

	// 空输入 - 返回空 map 而非 nil
	result := tc.GetUniqIdsByCustomerIds(nil)
	if result == nil {
		t.Error("GetUniqIdsByCustomerIds(nil) returned nil, want empty map")
	}

	result = tc.GetUniqIdsByCustomerIds([]string{})
	if result == nil || len(result) != 0 {
		t.Errorf("GetUniqIdsByCustomerIds([]) = %v, want empty map", result)
	}

	// 添加数据
	tc.Set("customer1", "uniq1")
	tc.Set("customer1", "uniq2")
	tc.Set("customer2", "uniq3")
	tc.Set("customer3", "uniq4")
	tc.Set("customer3", "uniq5")

	// 批量获取
	result = tc.GetUniqIdsByCustomerIds([]string{"customer1", "customer2"})
	if len(result) != 2 {
		t.Errorf("GetUniqIdsByCustomerIds returned %d customers, want 2", len(result))
	}

	if len(result["customer1"]) != 2 {
		t.Errorf("customer1 should have 2 uniqIds, got %d", len(result["customer1"]))
	}
	if len(result["customer2"]) != 1 {
		t.Errorf("customer2 should have 1 uniqId, got %d", len(result["customer2"]))
	}

	// 查询不存在的客户
	result = tc.GetUniqIdsByCustomerIds([]string{"nonexistent"})
	if len(result) != 0 {
		t.Errorf("GetUniqIdsByCustomerIds([nonexistent]) = %v, want empty map", result)
	}

	// 混合查询
	result = tc.GetUniqIdsByCustomerIds([]string{"customer1", "nonexistent"})
	if len(result) != 1 {
		t.Errorf("GetUniqIdsByCustomerIds([customer1, nonexistent]) returned %d customers, want 1", len(result))
	}
	if len(result["customer1"]) != 2 {
		t.Errorf("customer1 should have 2 uniqIds, got %d", len(result["customer1"]))
	}

	// 包含空字符串
	result = tc.GetUniqIdsByCustomerIds([]string{"customer1", "", "customer2"})
	if len(result) != 2 {
		t.Errorf("GetUniqIdsByCustomerIds with empty string returned %d customers, want 2", len(result))
	}

	// 验证返回的是副本
	result["customer1"][0] = "modified"
	original := tc.GetUniqIdsByCustomerId("customer1")
	if original[0] == "modified" {
		t.Error("modifying returned map values should not affect internal data")
	}
}

// TestConcurrentAccess 测试并发访问的安全性
func TestConcurrentAccess(t *testing.T) {
	tc := newTestCollect()
	var wg sync.WaitGroup

	concurrentCustomers := 100
	uniqIdsPerCustomer := 10

	// 并发写入
	for i := 0; i < concurrentCustomers; i++ {
		wg.Add(1)
		go func(customerIdx int) {
			defer wg.Done()
			customerId := "customer" + string(rune('A'+customerIdx%26))
			for j := 0; j < uniqIdsPerCustomer; j++ {
				uniqId := "uniq" + string(rune(customerIdx)) + string(rune(j))
				tc.Set(customerId, uniqId)
			}
		}(i)
	}
	wg.Wait()

	// 并发读取
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = tc.Len()
			_ = tc.GetCustomerIds()
			customerIds := tc.GetCustomerIds()
			if len(customerIds) > 0 {
				_ = tc.GetUniqIdsByCustomerId(customerIds[0])
				_ = tc.GetUniqIdsByCustomerIds(customerIds)
			}
		}()
	}
	wg.Wait()

	// 并发删除
	for i := 0; i < concurrentCustomers; i++ {
		wg.Add(1)
		go func(customerIdx int) {
			defer wg.Done()
			customerId := "customer" + string(rune('A'+customerIdx%26))
			for j := 0; j < uniqIdsPerCustomer; j++ {
				uniqId := "uniq" + string(rune(customerIdx)) + string(rune(j))
				tc.Del(customerId, uniqId)
			}
		}(i)
	}
	wg.Wait()

	// 最终应该为空
	if tc.Len() != 0 {
		t.Errorf("after all deletions, Len() = %d, want 0", tc.Len())
	}
}

// TestShardingDistribution 测试分片分布的均匀性
func TestShardingDistribution(t *testing.T) {
	tc := newTestCollect()

	// 添加大量客户ID
	customerCount := 1000
	for i := 0; i < customerCount; i++ {
		customerId := "customer_" + string(rune(i))
		tc.Set(customerId, "uniq_"+string(rune(i)))
	}

	// 统计每个分片的客户数量
	shardCounts := make([]int, shardCount)
	for i := range tc.shards {
		tc.shards[i].mux.RLock()
		shardCounts[i] = len(tc.shards[i].data)
		tc.shards[i].mux.RUnlock()
	}

	// 计算平均值
	avg := float64(customerCount) / shardCount

	// 检查分布是否相对均匀（允许一定偏差）
	maxDeviation := 0
	for _, count := range shardCounts {
		deviation := count - int(avg)
		if deviation < 0 {
			deviation = -deviation
		}
		if deviation > maxDeviation {
			maxDeviation = deviation
		}
	}

	// 最大偏差不应超过平均值的 100%（FNV哈希在小样本下可能有波动）
	maxAllowedDeviation := int(avg * 1.0)
	if maxAllowedDeviation < 3 {
		maxAllowedDeviation = 3 // 至少允许3个偏差
	}
	if maxDeviation > maxAllowedDeviation {
		t.Errorf("shard distribution too uneven: maxDeviation=%d, avg=%.2f, maxAllowed=%d",
			maxDeviation, avg, maxAllowedDeviation)
	}

	t.Logf("Shard distribution: avg=%.2f, maxDeviation=%d, total=%d", avg, maxDeviation, customerCount)
}

// TestMemoryShrinkage 测试删除后的内存缩容
func TestMemoryShrinkage(t *testing.T) {
	tc := newTestCollect()

	// 添加多个 uniqId
	customerId := "customer1"
	for i := 0; i < 100; i++ {
		tc.Set(customerId, "uniq"+string(rune(i)))
	}

	// 获取初始容量
	tc.shards[hashCustomerId(customerId)].mux.RLock()
	initialCap := cap(tc.shards[hashCustomerId(customerId)].data[customerId])
	tc.shards[hashCustomerId(customerId)].mux.RUnlock()

	// 删除大部分 uniqId
	for i := 0; i < 90; i++ {
		tc.Del(customerId, "uniq"+string(rune(i)))
	}

	// 检查是否触发缩容
	tc.shards[hashCustomerId(customerId)].mux.RLock()
	finalSlice := tc.shards[hashCustomerId(customerId)].data[customerId]
	finalCap := cap(finalSlice)
	finalLen := len(finalSlice)
	tc.shards[hashCustomerId(customerId)].mux.RUnlock()

	// 如果容量 > 2*长度，应该已经缩容
	if finalCap > finalLen*2 {
		t.Errorf("slice should have been shrunk: cap=%d, len=%d", finalCap, finalLen)
	}

	t.Logf("Memory shrinkage test: initialCap=%d, finalCap=%d, finalLen=%d", initialCap, finalCap, finalLen)
}

// BenchmarkSet 性能基准测试
func BenchmarkSet(b *testing.B) {
	tc := newTestCollect()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		customerId := "customer" + string(rune(i%100))
		uniqId := "uniq" + string(rune(i))
		tc.Set(customerId, uniqId)
	}
}

// BenchmarkDel 性能基准测试
func BenchmarkDel(b *testing.B) {
	tc := newTestCollect()

	// 准备数据
	for i := 0; i < 1000; i++ {
		customerId := "customer" + string(rune(i%100))
		uniqId := "uniq" + string(rune(i))
		tc.Set(customerId, uniqId)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		customerId := "customer" + string(rune(i%100))
		uniqId := "uniq" + string(rune(i))
		tc.Del(customerId, uniqId)
	}
}

// BenchmarkGetUniqIdsByCustomerId 性能基准测试
func BenchmarkGetUniqIdsByCustomerId(b *testing.B) {
	tc := newTestCollect()

	// 准备数据
	for i := 0; i < 100; i++ {
		tc.Set("benchmark_customer", "uniq"+string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.GetUniqIdsByCustomerId("benchmark_customer")
	}
}

// BenchmarkGetUniqIdsByCustomerIds 性能基准测试
func BenchmarkGetUniqIdsByCustomerIds(b *testing.B) {
	tc := newTestCollect()
	customerIds := []string{"customer1", "customer2", "customer3", "customer4", "customer5"}

	// 准备数据
	for _, customerId := range customerIds {
		for i := 0; i < 20; i++ {
			tc.Set(customerId, "uniq"+string(rune(i)))
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.GetUniqIdsByCustomerIds(customerIds)
	}
}

// BenchmarkGetCustomerIds 性能基准测试
func BenchmarkGetCustomerIds(b *testing.B) {
	tc := newTestCollect()

	// 准备数据
	for i := 0; i < 100; i++ {
		tc.Set("customer"+string(rune(i)), "uniq"+string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.GetCustomerIds()
	}
}
