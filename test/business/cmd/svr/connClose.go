package svr

import (
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/userDb"
)

// ConnClose 客户端关闭连接
func ConnClose(param []byte, _ *connProcessor.ConnProcessor) {
	payload := internalProtocol.ConnClose{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal connClose.ConnClose error:%v", err)
		return
	}
	//解析网关中存储的用户信息
	user := userDb.ParseNetSvrInfo(payload.UserInfo)
	if user != nil {
		//用户关闭连接，更新数据库中的session id为0
		userDb.Collect.SetSessionId(user.Id, 0)
	}
}
