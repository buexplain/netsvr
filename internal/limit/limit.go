package limit

import (
	"golang.org/x/time/rate"
	"netsvr/configs"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
	"os"
)

type limiter interface {
	Allow() bool
}

type nilLimit struct {
}

func (nilLimit) Allow() bool {
	return true
}

type manager [workerManager.MaxWorkerId + 1]limiter

func (r manager) Allow(workerId int) bool {
	if workerId < workerManager.MinWorkerId || workerId > workerManager.MaxWorkerId {
		return false
	}
	return r[workerId].Allow()
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
