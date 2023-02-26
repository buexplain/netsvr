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
	//检查服务编号是否在允许的范围内
	if workerManager.MinWorkerId > payload.Id || payload.Id > workerManager.MaxWorkerId {
		logging.Error("WorkerId %d not in range: %d ~ %d", payload.Id, workerManager.MinWorkerId, workerManager.MaxWorkerId)
		processor.ForceClose()
		return
	}
	//检查当前的business连接是否已经注册过服务编号了，不允许重复注册
	if processor.GetWorkerId() > 0 {
		processor.ForceClose()
		logging.Error("WorkerId %d duplicate register are not allowed", 1)
		return
	}
	//设置business连接的服务编号
	processor.SetWorkerId(int(payload.Id))
	//将该服务编号登记到worker管理器中
	workerManager.Manager.Set(processor.GetWorkerId(), processor)
	//判断该business连接是否处理客户连接打开信息
	if payload.ProcessConnClose {
		workerManager.SetProcessConnCloseWorkerId(payload.Id)
	}
	//判断该business连接是否处理客户连接关闭信息
	if payload.ProcessConnOpen {
		workerManager.SetProcessConnOpenWorkerId(payload.Id)
	}
	//判断该business连接是否要开启更多的协程去处理它发来的请求命令
	if payload.ProcessCmdGoroutineNum > 1 {
		var i uint32 = 1
		for ; i < payload.ProcessCmdGoroutineNum; i++ {
			go processor.LoopCmd()
		}
	}
	logging.Debug("Register a business by id: %d", payload.Id)
}
