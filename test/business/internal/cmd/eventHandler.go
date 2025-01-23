package cmd

import (
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
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
