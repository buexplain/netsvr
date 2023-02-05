package protocol

import (
	"encoding/json"
	"github.com/lesismal/nbio/logging"
)

type Cmd int32

func (r Cmd) String() string {
	return cmdName[r]
}

// 客户端Cmd路由
const (
	RouterRespConnOpen Cmd = iota + 1
	RouterNetSvrStatus
	RouterTotalSessionId
	RouterTopicList
	RouterLogin
	RouterLogout
	RouterUpdateSessionUserInfo
	RouterForceOffline
	RouterSingleCast
	RouterMulticast
	RouterBroadcast
	RouterSubscribe
	RouterUnsubscribe
	RouterTopicsConnCount
	RouterTopicsSessionId
	RouterPublish
)

var cmdName = map[Cmd]string{
	RouterRespConnOpen:          "RouterRespConnOpen",          //响应连接成功信息
	RouterNetSvrStatus:          "RouterNetSvrStatus",          //获取网关的信息
	RouterTotalSessionId:        "RouterTotalSessionId",        //获取网关所有在线的session id
	RouterTopicList:             "RouterTopicList",             //获取已订阅的主题列表
	RouterLogin:                 "RouterLogin",                 //登录
	RouterLogout:                "RouterLogout",                //退出登录
	RouterUpdateSessionUserInfo: "RouterUpdateSessionUserInfo", //更新网关中的用户信息
	RouterForceOffline:          "RouterForceOffline",          //某个session id的连接强制关闭
	RouterSingleCast:            "RouterSingleCast",            //单播给某个用户
	RouterMulticast:             "RouterMulticast",             //组播给多个用户
	RouterBroadcast:             "RouterBroadcast",             //广播给所有用户
	RouterSubscribe:             "RouterSubscribe",             //订阅
	RouterUnsubscribe:           "RouterUnsubscribe",           //取消订阅
	RouterTopicsConnCount:       "RouterTopicsConnCount",       //获取网关中的某几个主题的连接数
	RouterTopicsSessionId:       "RouterTopicsSessionId",       //获取网关中的某几个主题的连接session id
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

// ForceOffline 强制踢下线某个用户
type ForceOffline struct {
	UserId int `json:"userId"`
}

// SingleCast 客户端发送的单播信息
type SingleCast struct {
	Message string
	UserId  int `json:"userId"`
}

// Multicast 客户端发送的组播信息
type Multicast struct {
	Message string
	UserIds []int `json:"userIds"`
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

// TopicsSessionId 获取网关中的某几个主题的连接session id
type TopicsSessionId struct {
	Topics []string
}

// TopicsConnCount 获取网关中的某几个主题的连接数
type TopicsConnCount struct {
	GetAll bool
	Topics []string
}

// Publish 客户端发送的发布信息
type Publish struct {
	Message string
	Topic   string
}
