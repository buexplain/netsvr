/**
* Copyright 2022 buexplain@qq.com
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
// 限制客户连接的打开速度
// 限制客户消息的转发速度
package limit

import (
	"github.com/buexplain/netsvr-protocol-go/constant"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/protocol"
	"golang.org/x/time/rate"
	"netsvr/configs"
	"netsvr/internal/log"
	"os"
)

type limiter interface {
	Allow() bool
	SetLimit(newLimit rate.Limit)
}

// 空壳子限流器
type nilLimit struct {
}

func (nilLimit) Allow() bool {
	return true
}

func (nilLimit) SetLimit(_ rate.Limit) {
}

type manager [constant.WorkerIdMax + 1]limiter

func (r manager) Allow(workerId int) bool {
	if workerId < constant.WorkerIdMin || workerId > constant.WorkerIdMax {
		return false
	}
	return r[workerId].Allow()
}

// SetLimits 给一批worker id设置新的限流值
func (r manager) SetLimits(num int32, workerIds []int32) {
	if num <= 0 {
		//无效参数，不予处理
		return
	}
	notes := map[*rate.Limiter]struct{}{}
	for _, workerId := range workerIds {
		//无效参数，不予处理
		if workerId < constant.WorkerIdMin || workerId > constant.WorkerIdMax {
			continue
		}
		l := r[workerId]
		//是个空壳子限流器，不予处理
		if _, ok := l.(nilLimit); ok {
			continue
		}
		if rl, ok := l.(*rate.Limiter); ok {
			//跳过已经设置过的限流器
			if _, ok = notes[rl]; ok {
				continue
			}
			//设置新地限流值
			rl.SetLimit(rate.Limit(num))
			notes[rl] = struct{}{}
		}
	}
}

func (r manager) Count() []*netsvrProtocol.LimitCountItem {
	notes := map[*rate.Limiter]*netsvrProtocol.LimitCountItem{}
	for workerId, l := range r {
		//是个空壳子限流器，不予处理
		if _, ok := l.(nilLimit); ok {
			continue
		}
		//处理真正地限流器
		rl, ok := l.(*rate.Limiter)
		if !ok {
			continue
		}
		if _, ok = notes[rl]; !ok {
			notes[rl] = &netsvrProtocol.LimitCountItem{Num: int32(rl.Limit()), WorkerIds: make([]int32, 0)}
		}
		notes[rl].WorkerIds = append(notes[rl].WorkerIds, int32(workerId))
	}
	items := make([]*netsvrProtocol.LimitCountItem, 0, len(notes))
	for _, v := range notes {
		items = append(items, v)
	}
	return items
}

var Manager manager

func init() {
	Manager = manager{}
	//循环处理每一个限流配置
	for _, v := range configs.Config.Limit {
		var l limiter
		for workerId := v.Min; workerId <= v.Max; workerId++ {
			if workerId < constant.WorkerIdMin || workerId > constant.WorkerIdMax {
				log.Logger.Error().Int("workerId", workerId).Int("workerIdMin", constant.WorkerIdMin).Int("workerIdMax", constant.WorkerIdMax).Msg("Limit workerId range overflow")
				os.Exit(1)
			}
			if v.Num <= 0 {
				continue
			}
			//如果这个workerId已经被初始化过，则说明配置的限流范围冲突了
			if Manager[workerId] != nil {
				log.Logger.Error().Str("conflictConfig", v.String()).Msg("Limit workerId range coincidence")
				os.Exit(1)
			}
			if l == nil {
				l = rate.NewLimiter(rate.Limit(v.Num), v.Num)
			}
			//同一个范围的workerId使用同一个限流器
			Manager[workerId] = l
		}
		l = nil
	}
	//填充满整个数组
	l := nilLimit{}
	for i := constant.WorkerIdMin; i <= constant.WorkerIdMax; i++ {
		if Manager[i] == nil {
			Manager[i] = l
		}
	}
	for _, v := range Manager.Count() {
		log.Logger.Debug().Ints32("workerIds", v.WorkerIds).Int32("limit", v.Num).Msg("limit config")
	}
}
