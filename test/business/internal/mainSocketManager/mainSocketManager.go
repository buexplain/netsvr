package mainSocketManager

import (
	"github.com/buexplain/netsvr-business-go/v2/contract"
	"github.com/buexplain/netsvr-business-go/v2/mainSocket"
	"github.com/buexplain/netsvr-business-go/v2/socket"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"netsvr/test/business/configs"
	"netsvr/test/business/internal/cmd"
	"time"
)

var MainSocketManager *mainSocket.Manager

type emptyEventHandler struct {
}

func (r *emptyEventHandler) OnOpen(*netsvrProtocol.ConnOpen) {
}
func (r *emptyEventHandler) OnMessage(*netsvrProtocol.Transfer) {
}
func (r *emptyEventHandler) OnClose(*netsvrProtocol.ConnClose) {
}

func init() {
	MainSocketManager = mainSocket.NewManager()
	var eh contract.EventInterface
	if configs.Config.Service == "worker" {
		//以worker提供服务，则使用cmd.EventHandler
		eh = cmd.EventHandler
	} else {
		//反之使用空事件处理
		eh = &emptyEventHandler{}
	}
	sk := mainSocket.New(
		eh,
		socket.New(configs.Config.WorkerListenAddress, 0, 30*time.Second, 30*time.Second),
		configs.Config.WorkerHeartbeatMessage,
		netsvrProtocol.Event_OnOpen|netsvrProtocol.Event_OnClose|netsvrProtocol.Event_OnMessage,
		time.Second*45,
	)
	MainSocketManager.AddSocket(sk)
}
