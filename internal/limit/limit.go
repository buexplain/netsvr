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

// Package limit 限流模块
// 限制客户消息的转发速度
package limit

import (
	"golang.org/x/time/rate"
)

type limiter interface {
	Allow() bool
	Limit() rate.Limit
	SetLimit(newLimit rate.Limit)
}

// 空壳子限流器
type nilLimit struct {
}

func (nilLimit) Allow() bool {
	return true
}

func (nilLimit) Limit() rate.Limit {
	return 0
}
func (nilLimit) SetLimit(_ rate.Limit) {
}
