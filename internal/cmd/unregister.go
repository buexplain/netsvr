package cmd

import (
	"github.com/lesismal/nbio/logging"
	workerManager "netsvr/internal/worker/manager"
)

// Unregister business取消已注册的服务编号
func Unregister(_ []byte, processor *workerManager.ConnProcessor) {
	workerId := processor.GetWorkerId()
	workerManager.Manager.Del(workerId, processor)
	logging.Debug("Unregister a business by id: %d", workerId)
}
