package mainSocketManager

import (
	"github.com/buexplain/netsvr-business-go/v2/mainSocket"
	"github.com/buexplain/netsvr-business-go/v2/socket"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"netsvr/test/business/configs"
	"netsvr/test/business/internal/cmd"
	"time"
)

var MainSocketManager *mainSocket.Manager

func init() {
	MainSocketManager = mainSocket.NewManager()
	sk := mainSocket.New(
		cmd.EventHandler,
		socket.New(configs.Config.WorkerListenAddress, 0, 30*time.Second, 30*time.Second),
		configs.Config.WorkerHeartbeatMessage,
		netsvrProtocol.Event_OnOpen|netsvrProtocol.Event_OnClose|netsvrProtocol.Event_OnMessage,
		time.Second*45,
	)
	MainSocketManager.AddSocket(sk)
}
