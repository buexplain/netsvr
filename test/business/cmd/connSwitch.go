package cmd

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/userDb"
	"netsvr/test/protocol"
	"netsvr/test/utils"
)

type connSwitch struct{}

var ConnSwitch = connSwitch{}

func (r connSwitch) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterWorkerCmd(internalProtocol.Cmd_ConnOpen, r.ConnOpen)
	processor.RegisterWorkerCmd(internalProtocol.Cmd_ConnClose, r.ConnClose)
}

// ConnOpen 客户端打开连接
func (connSwitch) ConnOpen(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.ConnOpen{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.ConnOpen error:%v", err)
		return
	}
	//构造单播数据
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = payload.UniqId
	ret.Data = utils.NewResponse(protocol.RouterRespConnOpen, map[string]interface{}{
		"code":    0,
		"message": "连接网关成功",
		"data":    payload.UniqId,
	})
	//发送到网关
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ConnClose 客户端关闭连接
func (connSwitch) ConnClose(param []byte, _ *connProcessor.ConnProcessor) {
	payload := internalProtocol.ConnClose{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.ConnClose error: %v", err)
		return
	}
	//解析网关中存储的用户信息
	user := userDb.ParseNetSvrInfo(payload.Session)
	if user != nil {
		//更新数据库，标记用户已经下线
		userDb.Collect.SetOnline(user.Id, false)
	}
}
