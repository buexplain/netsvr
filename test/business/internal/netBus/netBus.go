package netBus

import (
	netsvrBusiness "github.com/buexplain/netsvr-business-go/v2"
	"github.com/buexplain/netsvr-business-go/v2/taskSocket"
	"netsvr/test/business/configs"
	"time"
)

var NetBus *netsvrBusiness.NetBus

func init() {
	poolManger := taskSocket.NewManger()
	factory := taskSocket.NewFactory(configs.Config.TaskListenAddress, time.Second*10, time.Second*10, time.Second*10)
	pool := taskSocket.NewPool(200, factory, time.Second*30, time.Second*10, configs.Config.TaskHeartbeatMessage)
	pool.LoopHeartbeat()
	poolManger.AddSocket(pool)
	NetBus = netsvrBusiness.NewNetBus(poolManger)
}
