// Package limit 限流模块
// 限制客户连接的打开速度
// 限制客户消息的转发速度
package limit

import (
	"golang.org/x/time/rate"
	"netsvr/configs"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/protocol"
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

type manager [workerManager.MaxWorkerId + 1]limiter

func (r manager) Allow(workerId int) bool {
	if workerId < workerManager.MinWorkerId || workerId > workerManager.MaxWorkerId {
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
		if workerId < workerManager.MinWorkerId || workerId > workerManager.MaxWorkerId {
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

func (r manager) Count() []*protocol.LimitCountItem {
	notes := map[*rate.Limiter]*protocol.LimitCountItem{}
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
			notes[rl] = &protocol.LimitCountItem{Num: int32(rl.Limit()), WorkerIds: make([]int32, 0)}
		}
		notes[rl].WorkerIds = append(notes[rl].WorkerIds, int32(workerId))
	}
	items := make([]*protocol.LimitCountItem, 0, len(notes))
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
			if workerId < workerManager.MinWorkerId || workerId > workerManager.MaxWorkerId {
				log.Logger.Error().Int("workerId", workerId).Int("minWorkerId", workerManager.MinWorkerId).Int("maxWorkerId", workerManager.MaxWorkerId).Msg("Limit range overflow")
				os.Exit(1)
			}
			if v.Num <= 0 {
				continue
			}
			//如果这个workerId已经被初始化过，则说明配置的限流范围冲突了
			if Manager[workerId] != nil {
				log.Logger.Error().Str("conflictConfig", v.String()).Msg("Limit range coincidence")
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
	for i := workerManager.MinWorkerId; i <= workerManager.MaxWorkerId; i++ {
		if Manager[i] == nil {
			Manager[i] = l
		}
	}
}
