package cmd

import (
	"github.com/lesismal/nbio/logging"
	workerManager "netsvr/internal/worker/manager"
)

// UnregisterWorker 取消已注册的工作进程的id，取消后不会再收到用户连接的转发信息
func UnregisterWorker(_ []byte, processor *workerManager.ConnProcessor) {
	workerId := processor.GetWorkerId()
	workerManager.Manager.Del(workerId, processor)
	logging.Debug("Unregister a worker by id: %d", workerId)
}
