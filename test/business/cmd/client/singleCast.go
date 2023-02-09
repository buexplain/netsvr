package client

import (
	"encoding/json"
	"fmt"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/userDb"
	workerUtils "netsvr/test/business/utils"
	"strconv"
)

// SingleCastForUserId 单播个某个用户
func SingleCastForUserId(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := protocol.SingleCastForUserId{}
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse protocol.SingleCastForUserId error: %v", err)
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	ret := &internalProtocol.SingleCast{}
	user := userDb.Collect.GetUserById(payload.UserId)
	if user == nil {
		//目标用户不存在，返回信息给到发送者
		ret.UniqId = tf.UniqId
		ret.Data = workerUtils.NewResponse(protocol.RouterSingleCastForUserId, map[string]interface{}{"code": 1, "message": "未找到目标用户"})
	} else {
		//目标用户存在，将信息转发给目标用户
		ret.UniqId = strconv.Itoa(user.Id)
		msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
		ret.Data = workerUtils.NewResponse(protocol.RouterSingleCastForUserId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// SingleCastForUniqId 单播给某个uniqId
func SingleCastForUniqId(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := protocol.SingleCastForUniqId{}
	if err := json.Unmarshal(workerUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse protocol.SingleCastForUniqId request error: %v", err)
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	//构建单播数据
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = payload.UniqId
	msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
	ret.Data = workerUtils.NewResponse(protocol.RouterSingleCastForUniqId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	//发到网关
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
