package cmd

import (
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
)

type EventHandler struct {
}

func (e EventHandler) OnOpen(connOpen *netsvrProtocol.ConnOpen) {
	ConnSwitch.ConnOpen(connOpen)
}

func (e EventHandler) OnMessage(transfer *netsvrProtocol.Transfer) {

}

func (e EventHandler) OnClose(connClose *netsvrProtocol.ConnClose) {
	ConnSwitch.ConnClose(connClose)
}
