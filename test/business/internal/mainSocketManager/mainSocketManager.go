package mainSocketManager

import (
	netsvrBusiness "github.com/buexplain/netsvr-business-go"
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"netsvr/test/business/configs"
	"time"
)

var MainSocketManager *netsvrBusiness.MainSocketManager

func Init(eventHandler netsvrBusiness.EventInterface) {
	MainSocketManager = netsvrBusiness.NewMainSocketManager()
	socket := netsvrBusiness.NewSocket(configs.Config.WorkerListenAddress, 0, 30*time.Second, 30*time.Second)
	mainSocket := netsvrBusiness.NewMainSocket(
		eventHandler,
		socket,
		configs.Config.WorkerHeartbeatMessage,
		netsvrProtocol.Event_OnOpen|netsvrProtocol.Event_OnClose|netsvrProtocol.Event_OnMessage,
		configs.Config.ProcessCmdGoroutineNum,
		time.Second*45,
	)
	MainSocketManager.AddSocket(mainSocket)
}
