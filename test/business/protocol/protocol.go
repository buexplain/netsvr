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
	RouterRespConnClose
	RouterMetrics
	RouterUniqIdList
	RouterUniqIdCount
	RouterTopicMyList
	RouterTopicCount
	RouterTopicList
	RouterSignIn
	RouterSignOut
	RouterCheckOnlineForUniqId
	RouterForceOfflineForUserId
	RouterForceOfflineForUniqId
	RouterSingleCastForUserId
	RouterSingleCastForUniqId
	RouterMulticastForUserId
	RouterMulticastForUniqId
	RouterBroadcast
	RouterTopicSubscribe
	RouterTopicUnsubscribe
	RouterTopicDelete
	RouterTopicUniqIdCount
	RouterTopicUniqIdList
	RouterTopicPublish
)

var CmdName = map[Cmd]string{
	RouterRespConnOpen:          "RouterRespConnOpen",          //响应连接成功信息
	RouterRespConnClose:         "RouterRespConnClose",         //响应连接成功信息
	RouterMetrics:               "RouterMetrics",               //获取网关统计的服务状态
	RouterUniqIdList:            "RouterUniqIdList",            //获取网关中全部的uniqId
	RouterUniqIdCount:           "RouterUniqIdCount",           //获取网关中uniqId的数量
	RouterTopicMyList:           "RouterTopicMyList",           //获取已订阅的主题列表
	RouterTopicCount:            "RouterTopicCount",            //获取网关中的主题
	RouterTopicList:             "RouterTopicList",             //获取网关中的主题数量
	RouterSignIn:                "RouterSignIn",                //登录
	RouterSignOut:               "RouterSignOut",               //退出登录
	RouterCheckOnlineForUniqId:  "RouterCheckOnlineForUniqId",  //检查某几个连接是否在线
	RouterForceOfflineForUserId: "RouterForceOfflineForUserId", //强制关闭某个连接
	RouterForceOfflineForUniqId: "RouterForceOfflineForUniqId", //强制关闭某个连接
	RouterSingleCastForUserId:   "RouterSingleCastForUserId",   //单播给某个用户
	RouterSingleCastForUniqId:   "RouterSingleCastForUniqId",   //单播给某个uniqId
	RouterMulticastForUserId:    "RouterMulticastForUserId",    //组播给多个用户
	RouterMulticastForUniqId:    "RouterMulticastForUniqId",    //组播给多个uniqId
	RouterBroadcast:             "RouterBroadcast",             //广播给所有用户
	RouterTopicSubscribe:        "RouterTopicSubscribe",        //订阅
	RouterTopicUnsubscribe:      "RouterTopicUnsubscribe",      //取消订阅
	RouterTopicDelete:           "RouterTopicDelete",           //删除主题
	RouterTopicUniqIdCount:      "RouterTopicUniqIdCount",      //获取网关中的某几个主题的连接数
	RouterTopicUniqIdList:       "RouterTopicUniqIdList",       //获取网关中的某个主题包含的uniqId
	RouterTopicPublish:          "RouterTopicPublish",          //发布信息
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
