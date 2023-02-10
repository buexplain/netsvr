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
		logging.Error("Proto unmarshal internalProtocol.ConnClose error:%v", err)
		return
	}
	//解析网关中存储的用户信息
	user := userDb.ParseNetSvrInfo(payload.Session)
	if user != nil {
		//更新数据库，标记用户已经下线
		userDb.Collect.SetOnline(user.Id, false)
	}
}
