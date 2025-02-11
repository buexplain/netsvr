package cmd

import (
	"encoding/json"
	netsvrBusiness "github.com/buexplain/netsvr-business-go"
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/pkg/protocol"
)

var businessCmdCallback map[protocol.Cmd]func(transfer *netsvrProtocol.Transfer, param string)
var EventHandler netsvrBusiness.EventInterface

func init() {
	businessCmdCallback = make(map[protocol.Cmd]func(transfer *netsvrProtocol.Transfer, param string))
	EventHandler = &eventHandler{}
}

type eventHandler struct {
}

func (e eventHandler) OnOpen(connOpen *netsvrProtocol.ConnOpen) {
	ConnSwitch.ConnOpen(connOpen)
}

func (e eventHandler) OnMessage(transfer *netsvrProtocol.Transfer) {
	//如果开始与结尾的字符不是花括号，说明不是有效的json字符串，则把数据原样echo回去
	if l := len(transfer.Data); l > 0 && (transfer.Data[0] == 123 && transfer.Data[l-1] == 125) == false {
		netBus.NetBus.SingleCast(transfer.UniqId, transfer.Data)
		return
	}
	clientRoute := new(protocol.ClientRouter)
	if err := json.Unmarshal(transfer.Data, clientRoute); err != nil {
		log.Logger.Debug().Err(err).
			Msg("Parse protocol.ClientRouter failed")
		return
	}
	log.Logger.Debug().
		Stringer("cmd", clientRoute.Cmd).
		Msg("Business receive client command")
	//客户发来的命令
	if callback, ok := businessCmdCallback[clientRoute.Cmd]; ok {
		callback(transfer, clientRoute.Data)
		return
	}
	//客户请求了错误的命令
	log.Logger.Debug().
		Interface("cmd", clientRoute.Cmd).
		Msg("Unknown protocol.clientRoute.Cmd")
}

func (e eventHandler) OnClose(connClose *netsvrProtocol.ConnClose) {
	ConnSwitch.ConnClose(connClose)
}
