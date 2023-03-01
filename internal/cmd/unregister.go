package cmd

import (
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// Unregister business取消已注册的服务编号
func Unregister(_ []byte, processor *workerManager.ConnProcessor) {
	workerId := processor.GetWorkerId()
	if workerManager.MinWorkerId <= workerId && workerId <= workerManager.MaxWorkerId {
		workerManager.Manager.Del(workerId, processor)
		log.Logger.Error().Int("workerId", workerId).Msg("Unregister a business")
	}
}
