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

package info

import (
	"netsvr/configs"
	"testing"
	"time"
)

// BenchmarkAllow
func BenchmarkAllow(b *testing.B) {
	info := New("7f00000117ac69d8f620af458912")

	b.ReportAllocs() //报告内存分配
	b.ResetTimer()   //排除初始化时间

	for i := 0; i < b.N; i++ {
		info.Allow()
	}
}

// TestAllowCorrectness 验证 Allow() 函数的正确性
func TestAllowCorrectness(t *testing.T) {
	// 保存原始配置
	originalMaxRequests := configs.Config.Customer.LimitWindowMaxRequests
	originalWindowSize := configs.Config.Customer.LimitWindowSizeSeconds
	defer func() {
		configs.Config.Customer.LimitWindowMaxRequests = originalMaxRequests
		configs.Config.Customer.LimitWindowSizeSeconds = originalWindowSize
	}()

	// 设置测试配置：窗口内最多允许 5 次请求
	configs.Config.Customer.LimitWindowMaxRequests = 5
	configs.Config.Customer.LimitWindowSizeSeconds = 60 // 60秒窗口，确保测试期间不会重置

	// 测试初始窗口（有10次免费请求）
	info := &Info{
		limitWindowStart:    time.Now().Unix(),
		limitWindowRequests: -10, // 初始有10次免费请求
	}

	allowedCount := 0
	for i := 0; i < 20; i++ {
		if info.Allow() {
			allowedCount++
		}
	}

	// 前10次是免费请求，接下来5次是窗口内的正常请求，总共15次
	expectedAllowed := int32(10) + configs.Config.Customer.LimitWindowMaxRequests
	if allowedCount != int(expectedAllowed) {
		t.Errorf("Expected %d allowed requests, got %d", expectedAllowed, allowedCount)
	}

	t.Logf("Initial window test passed: %d requests allowed (10 free + %d window)", allowedCount, configs.Config.Customer.LimitWindowMaxRequests)
}

// TestAllowWindowReset 验证窗口重置逻辑
func TestAllowWindowReset(t *testing.T) {
	// 保存原始配置
	originalMaxRequests := configs.Config.Customer.LimitWindowMaxRequests
	originalWindowSize := configs.Config.Customer.LimitWindowSizeSeconds
	defer func() {
		configs.Config.Customer.LimitWindowMaxRequests = originalMaxRequests
		configs.Config.Customer.LimitWindowSizeSeconds = originalWindowSize
	}()

	// 设置测试配置
	configs.Config.Customer.LimitWindowMaxRequests = 5
	configs.Config.Customer.LimitWindowSizeSeconds = 1 // 1秒窗口

	// 设置一个已过期的窗口（2秒前）
	pastTime := time.Now().Unix() - 2
	info := &Info{
		limitWindowStart:    pastTime,
		limitWindowRequests: 5, // 已达到限制
	}

	// 第一次调用应该触发窗口重置
	if !info.Allow() {
		t.Error("Expected request to be allowed after window reset")
	}

	// 验证窗口已重置
	if info.limitWindowRequests != 1 {
		t.Errorf("Expected limitWindowRequests to be 1 after reset, got %d", info.limitWindowRequests)
	}

	if info.limitWindowStart <= pastTime {
		t.Error("Expected limitWindowStart to be updated")
	}

	t.Log("Window reset test passed")
}
