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
	"netsvr/internal/utils/slicePool"
	"sync"
	"testing"
)

// newTestCollect 创建一个新的 collect 实例用于测试
func newTestCollect() *collect {
	c := &collect{}
	for i := range c.shards {
		c.shards[i].data = make(map[string]map[string]struct{}, 64)
	}
	return c
}

// TestHashTopic 测试哈希函数的一致性
func TestHashTopic(t *testing.T) {
	tests := []struct {
		name  string
		topic string
	}{
		{"empty", ""},
		{"simple", "topic1"},
		{"chinese", "主题测试"},
		{"special", "topic-with-special_chars!@#"},
		{"long", "this-is-a-very-long-topic-name-for-testing-purposes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashTopic(tt.topic)
			hash2 := hashTopic(tt.topic)
			// 同一输入应该产生相同输出
			if hash1 != hash2 {
				t.Errorf("hashTopic(%q) = %d, then %d, should be equal", tt.topic, hash1, hash2)
			}
			// 结果应该在有效范围内
			if hash1 < 0 || hash1 >= shardCount {
				t.Errorf("hashTopic(%q) = %d, should be in [0, %d)", tt.topic, hash1, shardCount)
			}
		})
	}
}

// TestLen 测试主题数量统计
func TestLen(t *testing.T) {
	tc := newTestCollect()

	// 初始状态应为 0
	if l := tc.Len(); l != 0 {
		t.Errorf("initial Len() = %d, want 0", l)
	}

	// 添加一些主题
	tc.SetBySlice([]string{"topic1", "topic2", "topic3"}, "user1")
	if l := tc.Len(); l != 3 {
		t.Errorf("after adding 3 topics, Len() = %d, want 3", l)
	}

	// 添加已存在的主题，数量不应变化
	tc.SetBySlice([]string{"topic1"}, "user2")
	if l := tc.Len(); l != 3 {
		t.Errorf("after adding existing topic, Len() = %d, want 3", l)
	}

	// 删除一个主题（只删除user1的订阅，但user2还在topic1中）
	topics := map[string]struct{}{"topic1": {}}
	tc.DelByMap(topics, "user1")
	// topic1仍然存在（因为user2还订阅着），所以总数还是3
	if l := tc.Len(); l != 3 {
		t.Errorf("after deleting user1 from topic1, Len() = %d, want 3 (topic1 still exists with user2)", l)
	}

	// 删除user2后，topic1应该被清理
	tc.DelByMap(topics, "user2")
	if l := tc.Len(); l != 2 {
		t.Errorf("after deleting user2 from topic1, Len() = %d, want 2", l)
	}
}

// TestCountAll 测试统计所有主题的订阅人数
func TestCountAll(t *testing.T) {
	tc := newTestCollect()

	// 空状态
	counts := tc.CountAll()
	if len(counts) != 0 {
		t.Errorf("CountAll() on empty = %v, want empty map", counts)
	}

	// 添加数据
	tc.SetBySlice([]string{"topic1", "topic2"}, "user1")
	tc.SetBySlice([]string{"topic1"}, "user2")
	tc.SetBySlice([]string{"topic2", "topic3"}, "user3")

	counts = tc.CountAll()
	if len(counts) != 3 {
		t.Errorf("CountAll() returned %d topics, want 3", len(counts))
	}

	expectedCounts := map[string]int32{
		"topic1": 2,
		"topic2": 2,
		"topic3": 1,
	}

	for topic, expected := range expectedCounts {
		if actual, ok := counts[topic]; !ok {
			t.Errorf("CountAll() missing topic %q", topic)
		} else if actual != expected {
			t.Errorf("CountAll()[%q] = %d, want %d", topic, actual, expected)
		}
	}
}

// TestCount 测试批量查询指定主题的订阅人数
func TestCount(t *testing.T) {
	tc := newTestCollect()

	// 空输入 - Count(nil) 返回 nil 是预期行为
	counts := tc.Count(nil)
	if counts != nil {
		t.Errorf("Count(nil) returned %v, want nil", counts)
	}

	counts = tc.Count([]string{})
	if len(counts) != 0 {
		t.Errorf("Count([]) = %v, want empty map", counts)
	}

	// 添加数据
	tc.SetBySlice([]string{"topic1", "topic2", "topic3"}, "user1")
	tc.SetBySlice([]string{"topic1", "topic2"}, "user2")

	// 查询存在的主题
	counts = tc.Count([]string{"topic1", "topic2"})
	if len(counts) != 2 {
		t.Errorf("Count() returned %d topics, want 2", len(counts))
	}
	if counts["topic1"] != 2 {
		t.Errorf("Count()[topic1] = %d, want 2", counts["topic1"])
	}
	if counts["topic2"] != 2 {
		t.Errorf("Count()[topic2] = %d, want 2", counts["topic2"])
	}

	// 查询不存在的主题
	counts = tc.Count([]string{"nonexistent"})
	if len(counts) != 0 {
		t.Errorf("Count([nonexistent]) = %v, want empty map", counts)
	}

	// 混合查询
	counts = tc.Count([]string{"topic1", "nonexistent"})
	if len(counts) != 1 {
		t.Errorf("Count([topic1, nonexistent]) returned %d topics, want 1", len(counts))
	}
	if counts["topic1"] != 2 {
		t.Errorf("Count()[topic1] = %d, want 2", counts["topic1"])
	}

	// 包含空字符串
	counts = tc.Count([]string{"topic1", "", "topic2"})
	if len(counts) != 2 {
		t.Errorf("Count with empty string returned %d topics, want 2", len(counts))
	}
}

// TestSetBySlice 测试设置订阅关系
func TestSetBySlice(t *testing.T) {
	tc := newTestCollect()

	// 空输入
	tc.SetBySlice(nil, "user1")
	tc.SetBySlice([]string{}, "user1")
	tc.SetBySlice([]string{"topic1"}, "")
	if tc.Len() != 0 {
		t.Error("SetBySlice with empty input should not add any topics")
	}

	// 正常添加
	tc.SetBySlice([]string{"topic1", "topic2"}, "user1")
	if tc.Len() != 2 {
		t.Errorf("after SetBySlice, Len() = %d, want 2", tc.Len())
	}

	// 验证订阅者
	sp := slicePool.NewStrSlice(10)

	uniqIds1 := tc.GetUniqIds("topic1", sp)
	if uniqIds1 == nil || len(*uniqIds1) != 1 {
		t.Errorf("topic1 should have 1 subscriber, got %v", uniqIds1)
	}
	sp.Put(uniqIds1)

	uniqIds2 := tc.GetUniqIds("topic2", sp)
	if uniqIds2 == nil || len(*uniqIds2) != 1 {
		t.Errorf("topic2 should have 1 subscriber, got %v", uniqIds2)
	}
	sp.Put(uniqIds2)

	// 添加新用户到已有主题
	tc.SetBySlice([]string{"topic1"}, "user2")
	uniqIds1 = tc.GetUniqIds("topic1", sp)
	if uniqIds1 == nil || len(*uniqIds1) != 2 {
		t.Errorf("topic1 should have 2 subscribers after adding user2, got %v", uniqIds1)
	}
	sp.Put(uniqIds1)

	// 重复添加同一用户（幂等性）
	tc.SetBySlice([]string{"topic1"}, "user2")
	uniqIds1 = tc.GetUniqIds("topic1", sp)
	if uniqIds1 == nil || len(*uniqIds1) != 2 {
		t.Errorf("topic1 should still have 2 subscribers after duplicate add, got %v", uniqIds1)
	}
	sp.Put(uniqIds1)

	// 包含空字符串的主题
	tc.SetBySlice([]string{"topic3", "", "topic4"}, "user3")
	if tc.Len() != 4 {
		t.Errorf("after SetBySlice with empty string, Len() = %d, want 4", tc.Len())
	}
}

// TestPullAndReturnUniqIds 测试删除主题并返回订阅者
func TestPullAndReturnUniqIds(t *testing.T) {
	tc := newTestCollect()

	// 空输入
	result := tc.PullAndReturnUniqIds(nil)
	if result != nil {
		t.Errorf("PullAndReturnUniqIds(nil) = %v, want nil", result)
	}

	result = tc.PullAndReturnUniqIds([]string{})
	// PullAndReturnUniqIds([]) 返回 nil 是预期行为（见源码162-163行）
	if result != nil {
		t.Errorf("PullAndReturnUniqIds([]) = %v, want nil", result)
	}

	// 添加数据
	tc.SetBySlice([]string{"topic1", "topic2"}, "user1")
	tc.SetBySlice([]string{"topic1"}, "user2")
	tc.SetBySlice([]string{"topic3"}, "user3")

	// 删除存在的主题
	result = tc.PullAndReturnUniqIds([]string{"topic1", "topic2"})
	if len(result) != 2 {
		t.Errorf("PullAndReturnUniqIds returned %d topics, want 2", len(result))
	}

	// 验证返回的订阅者
	if len(result["topic1"]) != 2 {
		t.Errorf("topic1 should have 2 subscribers, got %d", len(result["topic1"]))
	}
	if len(result["topic2"]) != 1 {
		t.Errorf("topic2 should have 1 subscriber, got %d", len(result["topic2"]))
	}

	// 验证主题已被删除
	if tc.Len() != 1 {
		t.Errorf("after PullAndReturnUniqIds, Len() = %d, want 1", tc.Len())
	}

	// 验证可以修改返回的 map（所有权转移）
	result["topic1"]["user3"] = struct{}{}
	if len(result["topic1"]) != 3 {
		t.Error("should be able to modify returned map")
	}

	// 删除不存在的主题
	result = tc.PullAndReturnUniqIds([]string{"nonexistent"})
	if len(result) != 0 {
		t.Errorf("PullAndReturnUniqIds([nonexistent]) = %v, want empty map", result)
	}

	// 包含空字符串
	tc.SetBySlice([]string{"topic4"}, "user4")
	result = tc.PullAndReturnUniqIds([]string{"topic4", ""})
	if len(result) != 1 {
		t.Errorf("PullAndReturnUniqIds with empty string returned %d topics, want 1", len(result))
	}
}

// TestGetUniqIds 测试获取单个主题的订阅者
func TestGetUniqIds(t *testing.T) {
	tc := newTestCollect()
	sp := slicePool.NewStrSlice(10)

	// 空主题
	result := tc.GetUniqIds("", sp)
	if result != nil {
		t.Errorf("GetUniqIds(\"\") = %v, want nil", result)
	}

	// 不存在的主题
	result = tc.GetUniqIds("nonexistent", sp)
	if result != nil {
		t.Errorf("GetUniqIds(nonexistent) = %v, want nil", result)
	}

	// 添加数据
	tc.SetBySlice([]string{"topic1"}, "user1")
	tc.SetBySlice([]string{"topic1"}, "user2")
	tc.SetBySlice([]string{"topic1"}, "user3")

	// 获取订阅者
	result = tc.GetUniqIds("topic1", sp)
	if result == nil {
		t.Fatal("GetUniqIds(topic1) returned nil")
	}
	if len(*result) != 3 {
		t.Errorf("GetUniqIds(topic1) returned %d users, want 3", len(*result))
	}

	// 验证包含所有用户
	userMap := make(map[string]bool)
	for _, uid := range *result {
		userMap[uid] = true
	}
	if !userMap["user1"] || !userMap["user2"] || !userMap["user3"] {
		t.Errorf("GetUniqIds(topic1) = %v, missing some users", *result)
	}

	// 归还切片到池
	sp.Put(result)

	// 再次使用切片池（验证复用）
	result2 := tc.GetUniqIds("topic1", sp)
	if result2 == nil {
		t.Fatal("GetUniqIds(topic1) second call returned nil")
	}
	sp.Put(result2)
}

// TestGetUniqIdsByTopics 测试批量获取多个主题的订阅者
func TestGetUniqIdsByTopics(t *testing.T) {
	tc := newTestCollect()

	// 空输入
	result := tc.GetUniqIdsByTopics(nil)
	if result != nil {
		t.Errorf("GetUniqIdsByTopics(nil) = %v, want nil", result)
	}

	result = tc.GetUniqIdsByTopics([]string{})
	// GetUniqIdsByTopics([]) 返回 nil 是预期行为（见源码218-219行）
	if result != nil {
		t.Errorf("GetUniqIdsByTopics([]) = %v, want nil", result)
	}

	// 添加数据
	tc.SetBySlice([]string{"topic1", "topic2"}, "user1")
	tc.SetBySlice([]string{"topic1"}, "user2")
	tc.SetBySlice([]string{"topic2", "topic3"}, "user3")

	// 批量获取
	result = tc.GetUniqIdsByTopics([]string{"topic1", "topic2"})
	if len(result) != 2 {
		t.Errorf("GetUniqIdsByTopics returned %d topics, want 2", len(result))
	}

	if len(result["topic1"]) != 2 {
		t.Errorf("topic1 should have 2 subscribers, got %d", len(result["topic1"]))
	}
	if len(result["topic2"]) != 2 {
		t.Errorf("topic2 should have 2 subscribers, got %d", len(result["topic2"]))
	}

	// 查询不存在的主题
	result = tc.GetUniqIdsByTopics([]string{"nonexistent"})
	if len(result) != 0 {
		t.Errorf("GetUniqIdsByTopics([nonexistent]) = %v, want empty map", result)
	}

	// 包含空字符串
	result = tc.GetUniqIdsByTopics([]string{"topic1", ""})
	if len(result) != 1 {
		t.Errorf("GetUniqIdsByTopics with empty string returned %d topics, want 1", len(result))
	}
}

// TestGet 测试获取所有主题列表
func TestGet(t *testing.T) {
	tc := newTestCollect()

	// 空状态
	topics := tc.Get()
	if len(topics) != 0 {
		t.Errorf("Get() on empty = %v, want empty slice", topics)
	}

	// 添加主题
	tc.SetBySlice([]string{"topic1", "topic2", "topic3"}, "user1")
	topics = tc.Get()
	if len(topics) != 3 {
		t.Errorf("Get() returned %d topics, want 3", len(topics))
	}

	// 验证包含所有主题
	topicMap := make(map[string]bool)
	for _, topic := range topics {
		topicMap[topic] = true
	}
	if !topicMap["topic1"] || !topicMap["topic2"] || !topicMap["topic3"] {
		t.Errorf("Get() = %v, missing some topics", topics)
	}
}

// TestDelByMap 测试通过 map 删除订阅关系
func TestDelByMap(t *testing.T) {
	tc := newTestCollect()

	// 空输入
	emptyMap := map[string]struct{}{}
	tc.DelByMap(emptyMap, "user1")
	if tc.Len() != 0 {
		t.Error("DelByMap with empty map should not affect anything")
	}

	// 添加数据
	tc.SetBySlice([]string{"topic1", "topic2", "topic3"}, "user1")
	tc.SetBySlice([]string{"topic1", "topic2"}, "user2")

	// 删除用户对某些主题的订阅
	topics := map[string]struct{}{
		"topic1": {},
		"topic2": {},
	}
	tc.DelByMap(topics, "user1")

	// 验证 user1 已取消订阅
	sp := slicePool.NewStrSlice(10)

	uniqIds1 := tc.GetUniqIds("topic1", sp)
	if uniqIds1 == nil || len(*uniqIds1) != 1 {
		t.Errorf("topic1 should have 1 subscriber after deletion, got %v", uniqIds1)
	}
	sp.Put(uniqIds1)

	uniqIds2 := tc.GetUniqIds("topic2", sp)
	if uniqIds2 == nil || len(*uniqIds2) != 1 {
		t.Errorf("topic2 should have 1 subscriber after deletion, got %v", uniqIds2)
	}
	sp.Put(uniqIds2)

	// topic3 应该还存在且只有 user1
	uniqIds3 := tc.GetUniqIds("topic3", sp)
	if uniqIds3 == nil || len(*uniqIds3) != 1 {
		t.Errorf("topic3 should have 1 subscriber, got %v", uniqIds3)
	}
	sp.Put(uniqIds3)

	// 删除最后一个订阅者，主题应被清理
	topics = map[string]struct{}{"topic3": {}}
	tc.DelByMap(topics, "user1")
	if tc.Len() != 2 {
		t.Errorf("after deleting last subscriber, Len() = %d, want 2", tc.Len())
	}

	// 删除不存在的主题
	topics = map[string]struct{}{"nonexistent": {}}
	tc.DelByMap(topics, "user1") // 不应该 panic
}

// TestDelBySlice 测试通过 slice 删除订阅关系
func TestDelBySlice(t *testing.T) {
	tc := newTestCollect()

	// 空输入
	tc.DelBySlice(nil, "user1")
	tc.DelBySlice([]string{}, "user1")
	if tc.Len() != 0 {
		t.Error("DelBySlice with empty input should not affect anything")
	}

	// 添加数据
	tc.SetBySlice([]string{"topic1", "topic2", "topic3"}, "user1")
	tc.SetBySlice([]string{"topic1", "topic2"}, "user2")

	// 删除用户对某些主题的订阅
	tc.DelBySlice([]string{"topic1", "topic2"}, "user1")

	// 验证 user1 已取消订阅
	sp := slicePool.NewStrSlice(10)

	uniqIds1 := tc.GetUniqIds("topic1", sp)
	if uniqIds1 == nil || len(*uniqIds1) != 1 {
		t.Errorf("topic1 should have 1 subscriber after deletion, got %v", uniqIds1)
	}
	sp.Put(uniqIds1)

	uniqIds2 := tc.GetUniqIds("topic2", sp)
	if uniqIds2 == nil || len(*uniqIds2) != 1 {
		t.Errorf("topic2 should have 1 subscriber after deletion, got %v", uniqIds2)
	}
	sp.Put(uniqIds2)

	// topic3 应该还存在且只有 user1
	uniqIds3 := tc.GetUniqIds("topic3", sp)
	if uniqIds3 == nil || len(*uniqIds3) != 1 {
		t.Errorf("topic3 should have 1 subscriber, got %v", uniqIds3)
	}
	sp.Put(uniqIds3)

	// 删除最后一个订阅者，主题应被清理
	tc.DelBySlice([]string{"topic3"}, "user1")
	if tc.Len() != 2 {
		t.Errorf("after deleting last subscriber, Len() = %d, want 2", tc.Len())
	}

	// 包含空字符串
	tc.SetBySlice([]string{"topic4"}, "user3")
	tc.DelBySlice([]string{"topic4", ""}, "user3")
	if tc.Len() != 2 {
		t.Errorf("after DelBySlice with empty string, Len() = %d, want 2", tc.Len())
	}
}

// TestConcurrentAccess 测试并发访问的安全性
func TestConcurrentAccess(t *testing.T) {
	tc := newTestCollect()
	var wg sync.WaitGroup

	concurrentUsers := 100
	topicsPerUser := 10

	// 并发写入
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func(userId int) {
			defer wg.Done()
			var topics []string
			for j := 0; j < topicsPerUser; j++ {
				topics = append(topics, "topic"+string(rune('A'+userId%26))+string(rune(j)))
			}
			tc.SetBySlice(topics, "user"+string(rune(userId)))
		}(i)
	}
	wg.Wait()

	// 并发读取
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = tc.Len()
			_ = tc.CountAll()
			_ = tc.Get()
		}()
	}
	wg.Wait()

	// 并发删除
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func(userId int) {
			defer wg.Done()
			var topics []string
			for j := 0; j < topicsPerUser; j++ {
				topics = append(topics, "topic"+string(rune('A'+userId%26))+string(rune(j)))
			}
			tc.DelBySlice(topics, "user"+string(rune(userId)))
		}(i)
	}
	wg.Wait()

	// 最终应该为空
	if tc.Len() != 0 {
		t.Errorf("after all deletions, Len() = %d, want 0", tc.Len())
	}
}

// BenchmarkSetBySlice 性能基准测试
func BenchmarkSetBySlice(b *testing.B) {
	tc := newTestCollect()
	topics := []string{"topic1", "topic2", "topic3", "topic4", "topic5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.SetBySlice(topics, "user"+string(rune(i%26)))
	}
}

// BenchmarkGetUniqIds 性能基准测试
func BenchmarkGetUniqIds(b *testing.B) {
	tc := newTestCollect()
	sp := slicePool.NewStrSlice(100)

	// 准备数据
	for i := 0; i < 100; i++ {
		tc.SetBySlice([]string{"benchmark_topic"}, "user"+string(rune(i%26)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := tc.GetUniqIds("benchmark_topic", sp)
		if result != nil {
			sp.Put(result)
		}
	}
}

// BenchmarkCount 性能基准测试
func BenchmarkCount(b *testing.B) {
	tc := newTestCollect()
	topics := []string{"topic1", "topic2", "topic3", "topic4", "topic5"}

	// 准备数据
	for i := 0; i < 100; i++ {
		tc.SetBySlice(topics, "user"+string(rune(i%26)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tc.Count(topics)
	}
}
