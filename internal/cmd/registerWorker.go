package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol/toServer/registerWorker"
	workerManager "netsvr/internal/worker/manager"
)

// RegisterWorker 注册工作进程
func RegisterWorker(param []byte, processor *workerManager.ConnProcessor) {
	req := registerWorker.RegisterWorker{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal registerWorker.RegisterWorker error: %v", err)
		return
	}
	if workerManager.MinWorkerId > req.Id || req.Id > workerManager.MaxWorkerId {
		logging.Error("WorkerId %d not in range: %d ~ %d", req.Id, workerManager.MinWorkerId, workerManager.MaxWorkerId)
		processor.Close()
		return
	}
	processor.SetWorkerId(int(req.Id))
	workerManager.Manager.Set(processor.GetWorkerId(), processor)
	if req.ProcessConnClose {
		workerManager.SetProcessConnCloseWorkerId(req.Id)
	}
	if req.ProcessConnOpen {
		workerManager.SetProcessConnOpenWorkerId(req.Id)
	}
	logging.Debug("Register a worker by id: %d", req.Id)
}
