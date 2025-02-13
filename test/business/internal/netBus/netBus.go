package netBus

import (
	netsvrBusiness "github.com/buexplain/netsvr-business-go"
	"netsvr/test/business/configs"
	"netsvr/test/business/internal/mainSocketManager"
	"time"
)

var NetBus *netsvrBusiness.NetBus

func Init() {
	poolManger := netsvrBusiness.NewTaskSocketPoolManger()
	factory := netsvrBusiness.NewTaskSocketFactory(configs.Config.WorkerListenAddress, time.Second*10, time.Second*10, time.Second*10)
	pool := netsvrBusiness.NewTaskSocketPool(configs.Config.ProcessCmdGoroutineNum, factory, time.Second*30, time.Second*10, configs.Config.WorkerHeartbeatMessage)
	pool.LoopHeartbeat()
	poolManger.AddSocket(pool)
	NetBus = netsvrBusiness.NewNetBus(mainSocketManager.MainSocketManager, poolManger)
}
