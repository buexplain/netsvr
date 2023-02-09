package protocol

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
)

type Cmd int32

func (r Cmd) String() string {
	return CmdName[r]
}

// 客户端Cmd路由
const (
	RouterRespConnOpen Cmd = iota + 1
	RouterNetSvrStatus
	RouterTotalUniqIds
	RouterTopicList
	RouterLogin
	RouterLogout
	RouterForceOfflineForUserId
	RouterSingleCastForUserId
	RouterSingleCastForUniqId
	RouterMulticastForUserId
	RouterMulticastForUniqId
	RouterBroadcast
	RouterSubscribe
	RouterUnsubscribe
	RouterTopicsUniqIdCount
	RouterTopicUniqIds
	RouterPublish
)

var CmdName = map[Cmd]string{
	RouterRespConnOpen:          "RouterRespConnOpen",          //响应连接成功信息
	RouterNetSvrStatus:          "RouterNetSvrStatus",          //获取网关的信息
	RouterTotalUniqIds:          "RouterTotalUniqIds",          //获取网关所有在线的session id
	RouterTopicList:             "RouterTopicList",             //获取已订阅的主题列表
	RouterLogin:                 "RouterLogin",                 //登录
	RouterLogout:                "RouterLogout",                //退出登录
	RouterForceOfflineForUserId: "RouterForceOfflineForUserId", //强制关闭某个连接
	RouterSingleCastForUserId:   "RouterSingleCastForUserId",   //单播给某个用户
	RouterSingleCastForUniqId:   "RouterSingleCastForUniqId",   //单播给某个session id
	RouterMulticastForUserId:    "RouterMulticastForUserId",    //组播给多个用户
	RouterMulticastForUniqId:    "RouterMulticastForUniqId",    //组播给多个uniqId
	RouterBroadcast:             "RouterBroadcast",             //广播给所有用户
	RouterSubscribe:             "RouterSubscribe",             //订阅
	RouterUnsubscribe:           "RouterUnsubscribe",           //取消订阅
	RouterTopicsUniqIdCount:     "RouterTopicsUniqIdCount",     //获取网关中的某几个主题的连接数
	RouterTopicUniqIds:          "RouterTopicUniqIds",          //获取网关中的某个主题包含的uniqId
	RouterPublish:               "RouterPublish",               //发布信息
}

type ClientRouter struct {
	Cmd  Cmd
	Data string
}

func ParseClientRouter(data []byte) *ClientRouter {
	ret := new(ClientRouter)
	if err := json.Unmarshal(data, ret); err != nil {
		logging.Debug("Parse client router error: %v", err)
		return nil
	}
	return ret
}

// Login 客户端发送的登录信息
type Login struct {
	Username string
	Password string
}

// ForceOfflineForUserId 强制踢下线某个用户
type ForceOfflineForUserId struct {
	UserId int `json:"userId"`
}

// SingleCastForUserId 客户端发送的单播信息
type SingleCastForUserId struct {
	Message string
	UserId  int `json:"userId"`
}

// SingleCastForUniqId 客户端发送的单播信息
type SingleCastForUniqId struct {
	Message string
	UniqId  string `json:"uniqId"`
}

// MulticastForUserId 客户端发送的组播信息
type MulticastForUserId struct {
	Message string
	UserIds []int `json:"userIds"`
}

// MulticastForUniqId 客户端发送的组播信息
type MulticastForUniqId struct {
	Message string
	UnIqIds []string `json:"unIqIds"`
}

// Broadcast 客户端发送的广播信息
type Broadcast struct {
	Message string
}

// Subscribe 客户端发送的订阅信息
type Subscribe struct {
	Topics []string
}

// Unsubscribe 客户端发送的取消订阅信息
type Unsubscribe struct {
	Topics []string
}

// TopicUniqIds 获取网关中的某个主题包含的uniqId
type TopicUniqIds struct {
	Topic string
}

// TopicsUniqIdCount 获取网关中的某几个主题的连接数
type TopicsUniqIdCount struct {
	CountAll bool
	Topics   []string
}

// Publish 客户端发送的发布信息
type Publish struct {
	Message string
	Topic   string
}
