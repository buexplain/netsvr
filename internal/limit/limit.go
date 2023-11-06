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
// 限制客户连接的打开速度
// 限制客户消息的转发速度
package limit

import (
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v2/netsvr"
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

type collect [netsvrProtocol.WorkerIdMax + 1]limiter

type manager struct {
	collect   collect
	nameIndex map[string]limiter
}

func (r manager) Allow(workerId int) bool {
	return r.collect[workerId].Allow()
}

// Update 更新限流器的并发设定
func (r manager) Update(concurrency int32, name string) {
	//无效参数，不予处理
	if concurrency <= 0 {
		return
	}
	//未找到限流器，不予处理
	l, ok := r.nameIndex[name]
	if !ok {
		return
	}
	l.SetLimit(rate.Limit(concurrency))
}

func (r manager) Count() []*netsvrProtocol.LimitRespItem {
	notes := map[*rate.Limiter]*netsvrProtocol.LimitRespItem{}
	for workerId, l := range r.collect {
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
			notes[rl] = &netsvrProtocol.LimitRespItem{Concurrency: int32(rl.Limit()), WorkerIds: make([]int32, 0)}
			for name, l := range r.nameIndex {
				if l == rl {
					notes[rl].Name = name
					break
				}
			}
		}
		notes[rl].WorkerIds = append(notes[rl].WorkerIds, int32(workerId))
	}
	items := make([]*netsvrProtocol.LimitRespItem, 0, len(notes))
	for _, v := range notes {
		items = append(items, v)
	}
	return items
}

var Manager manager

func init() {
	Manager = manager{collect: collect{}, nameIndex: map[string]limiter{}}
	//循环处理每一个限流配置
	for _, v := range configs.Config.Limit {
		//忽略无效的配置
		if v.Concurrency <= 0 || len(v.WorkerIds) == 0 {
			continue
		}
		l := rate.NewLimiter(rate.Limit(v.Concurrency), v.Concurrency)
		if Manager.nameIndex[v.Name] != nil {
			log.Logger.Error().Str("name", v.Name).Msg("Name is configured multiple times of limiter")
			os.Exit(1)
		}
		for _, workerId := range v.WorkerIds {
			if workerId < netsvrProtocol.WorkerIdMin || workerId > netsvrProtocol.WorkerIdMax {
				log.Logger.Error().Int("workerId", workerId).Int("workerIdMin", netsvrProtocol.WorkerIdMin).Int("workerIdMax", netsvrProtocol.WorkerIdMax).Msg("WorkerId range overflow of limiter")
				os.Exit(1)
			}
			//如果这个workerId已经被初始化过，则说明配置的限流范围冲突了，每个workerId只允许配置一次
			if Manager.collect[workerId] != nil {
				log.Logger.Error().Int("workerId", workerId).Msg("WorkerId is configured multiple times of limiter")
				os.Exit(1)
			}
			Manager.collect[workerId] = l
		}
		Manager.nameIndex[v.Name] = l
	}
	//填充满整个数组
	l := nilLimit{}
	for i := netsvrProtocol.WorkerIdMin; i <= netsvrProtocol.WorkerIdMax; i++ {
		if Manager.collect[i] == nil {
			Manager.collect[i] = l
		}
	}
	for _, v := range Manager.Count() {
		log.Logger.Debug().Str("name", v.Name).Int32("concurrency", v.Concurrency).Ints32("workerIds", v.WorkerIds).Msg("limit config")
	}
}
