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

// 本文件是与具体命令无关的通用断言。
package integration

import (
	"slices"
	"testing"
)

// requireOK 断言回包 code 为 0
func requireOK(t *testing.T, env *envelope) {
	t.Helper()
	if env.Code != 0 {
		t.Fatalf("业务进程返回失败：code=%d message=%s", env.Code, env.Message)
	}
}

// assertSameSet 忽略顺序比较两个集合（会去重）
func assertSameSet(t *testing.T, what string, want, got []string) {
	t.Helper()
	wantSorted := slices.Clone(want)
	gotSorted := slices.Clone(got)
	slices.Sort(wantSorted)
	slices.Sort(gotSorted)
	wantSorted = slices.Compact(wantSorted)
	gotSorted = slices.Compact(gotSorted)
	if !slices.Equal(wantSorted, gotSorted) {
		t.Fatalf("%s：期望 %v，实际 %v", what, want, got)
	}
}

// assertContains 断言集合里包含全部期望元素
func assertContains(t *testing.T, what string, want, got []string) {
	t.Helper()
	for _, item := range want {
		if !slices.Contains(got, item) {
			t.Fatalf("%s：期望包含 %v，实际 %v", what, want, got)
		}
	}
}
