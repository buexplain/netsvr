package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/protocol"
	"netsvr/test/business/userDb"
	businessUtils "netsvr/test/business/utils"
	"strconv"
)

type multicast struct{}

var Multicast = multicast{}

func (r multicast) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterMulticastForUserId, r.ForUserId)
	processor.RegisterBusinessCmd(protocol.RouterMulticastForUniqId, r.ForUniqId)
}

// MulticastForUserIdParam 客户端发送的组播信息
type MulticastForUserIdParam struct {
	Message string
	UserIds []int `json:"userIds"`
}

// ForUserId 组播给某几个用户
func (multicast) ForUserId(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(MulticastForUserIdParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse MulticastForUserIdParam error: %v", err)
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	//查询目标用户的
	userIds := make([]string, 0)
	for _, userId := range payload.UserIds {
		user := userDb.Collect.GetUserById(userId)
		if user.IsOnline {
			userIds = append(userIds, strconv.Itoa(user.Id))
		}
	}
	//构建组播数据
	ret := &internalProtocol.Multicast{}
	//没有找到任何目标用户，通知发送方，目标用户不存在
	if len(userIds) == 0 {
		//目标用户不存在，返回信息给到发送者
		userIds = append(userIds, tf.UniqId)
		ret.Data = businessUtils.NewResponse(protocol.RouterMulticastForUserId, map[string]interface{}{"code": 1, "message": "未找到目标用户"})
	} else {
		msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
		ret.Data = businessUtils.NewResponse(protocol.RouterMulticastForUserId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	}
	ret.UniqIds = userIds
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Multicast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// MulticastForUniqIdParam 客户端发送的组播信息
type MulticastForUniqIdParam struct {
	Message string
	UnIqIds []string `json:"unIqIds"`
}

// ForUniqId 组播给某几个uniqId
func (multicast) ForUniqId(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(MulticastForUniqIdParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse MulticastForUniqIdParam error: %v", err)
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	//构建组播数据
	ret := &internalProtocol.Multicast{}
	ret.UniqIds = payload.UnIqIds
	msg := map[string]interface{}{"fromUser": fromUser, "message": payload.Message}
	ret.Data = businessUtils.NewResponse(protocol.RouterMulticastForUniqId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	//发到网关
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Multicast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
