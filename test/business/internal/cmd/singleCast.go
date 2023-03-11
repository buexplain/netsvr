package cmd

import (
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/proto"
	netsvrProtocol "netsvr/pkg/protocol"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
	"strconv"
)

type singleCast struct{}

var SingleCast = singleCast{}

func (r singleCast) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterSingleCastForUserId, r.ForUserId)
	processor.RegisterBusinessCmd(protocol.RouterSingleCastForUniqId, r.ForUniqId)
}

// SingleCastForUserIdParam 客户端发送的单播信息
type SingleCastForUserIdParam struct {
	Message string
	UserId  int `json:"userId"`
}

// ForUserId 单播个某个用户
func (singleCast) ForUserId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := SingleCastForUserIdParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse SingleCastForUserIdParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	ret := &netsvrProtocol.SingleCast{}
	user := userDb.Collect.GetUserById(payload.UserId)
	if user == nil {
		//目标用户不存在，返回信息给到发送者
		ret.UniqId = tf.UniqId
		ret.Data = testUtils.NewResponse(protocol.RouterSingleCastForUserId, map[string]interface{}{"code": 1, "message": "未找到目标用户"})
	} else {
		//目标用户存在，将信息转发给目标用户
		ret.UniqId = strconv.Itoa(user.Id)
		msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
		ret.Data = testUtils.NewResponse(protocol.RouterSingleCastForUserId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	}
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// SingleCastForUniqIdParam 客户端发送的单播信息
type SingleCastForUniqIdParam struct {
	Message string
	UniqId  string `json:"uniqId"`
}

// ForUniqId 单播给某个uniqId
func (singleCast) ForUniqId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := SingleCastForUniqIdParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse SingleCastForUniqIdParam failed")
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
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = payload.UniqId
	msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
	ret.Data = testUtils.NewResponse(protocol.RouterSingleCastForUniqId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	//发到网关
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
