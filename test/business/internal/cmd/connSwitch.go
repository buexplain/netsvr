package cmd

import (
	"google.golang.org/protobuf/proto"
	netsvrProtocol "netsvr/pkg/protocol"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	"netsvr/test/pkg/utils"
)

type connSwitch struct{}

var ConnSwitch = connSwitch{}

func (r connSwitch) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterWorkerCmd(netsvrProtocol.Cmd_ConnOpen, r.ConnOpen)
	processor.RegisterWorkerCmd(netsvrProtocol.Cmd_ConnClose, r.ConnClose)
}

// ConnOpen 客户端打开连接
func (connSwitch) ConnOpen(param []byte, processor *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.ConnOpen{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.ConnOpen failed")
		return
	}
	//构造单播数据
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = payload.UniqId
	ret.Data = utils.NewResponse(protocol.RouterRespConnOpen, map[string]interface{}{
		"code":    0,
		"message": "连接网关成功",
		"data": map[string]interface{}{
			"uniqId":        payload.UniqId,
			"rawQuery":      payload.RawQuery,
			"subProtocol":   payload.SubProtocol,
			"xForwardedFor": payload.XForwardedFor,
			"xRealIP":       payload.XRealIP,
			"remoteIP":      payload.RemoteIP,
		},
	})
	//发送到网关
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ConnClose 客户端关闭连接
func (connSwitch) ConnClose(param []byte, _ *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.ConnClose{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.ConnClose failed")
		return
	}
	//解析网关中存储的用户信息
	user := userDb.ParseNetSvrInfo(payload.Session)
	if user != nil {
		//更新数据库，标记用户已经下线
		userDb.Collect.SetOnline(user.Id, false)
	}
}
