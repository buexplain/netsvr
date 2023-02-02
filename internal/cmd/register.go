package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// Register 注册business进程
func Register(param []byte, processor *workerManager.ConnProcessor) {
	payload := protocol.Register{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.Register error: %v", err)
		return
	}
	if workerManager.MinWorkerId > payload.Id || payload.Id > workerManager.MaxWorkerId {
		logging.Error("WorkerId %d not in range: %d ~ %d", payload.Id, workerManager.MinWorkerId, workerManager.MaxWorkerId)
		processor.Close()
		return
	}
	processor.SetWorkerId(int(payload.Id))
	workerManager.Manager.Set(processor.GetWorkerId(), processor)
	if payload.ProcessConnClose {
		workerManager.SetProcessConnCloseWorkerId(payload.Id)
	}
	if payload.ProcessConnOpen {
		workerManager.SetProcessConnOpenWorkerId(payload.Id)
	}
	logging.Debug("Register a business by id: %d", payload.Id)
}
