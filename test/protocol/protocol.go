package protocol

import (
	"encoding/json"
	"netsvr/internal/log"
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
	RouterLimit
	RouterUniqIdList
	RouterUniqIdCount
	RouterTopicMyList
	RouterTopicCount
	RouterTopicList
	RouterSignIn
	RouterSignOut
	RouterSignInForForge
	RouterSignOutForForge
	RouterCheckOnlineForUniqId
	RouterForceOfflineForUserId
	RouterForceOfflineForUniqId
	RouterForceOfflineGuestForUniqId
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
	RouterRespConnOpen:               "RouterRespConnOpen",               //响应连接成功信息
	RouterRespConnClose:              "RouterRespConnClose",              //响应连接成功信息
	RouterMetrics:                    "RouterMetrics",                    //获取网关状态的统计信息
	RouterLimit:                      "RouterLimit",                      //更新限流配置、获取网关中的限流配置的真实情况
	RouterUniqIdList:                 "RouterUniqIdList",                 //获取网关中全部的uniqId
	RouterUniqIdCount:                "RouterUniqIdCount",                //获取网关中uniqId的数量
	RouterTopicMyList:                "RouterTopicMyList",                //获取已订阅的主题列表
	RouterTopicCount:                 "RouterTopicCount",                 //获取网关中的主题
	RouterTopicList:                  "RouterTopicList",                  //获取网关中的主题数量
	RouterSignIn:                     "RouterSignIn",                     //登录
	RouterSignOut:                    "RouterSignOut",                    //退出登录
	RouterSignInForForge:             "RouterSignInForForge",             //伪造登录
	RouterSignOutForForge:            "RouterSignOutForForge",            //伪造退出登录
	RouterCheckOnlineForUniqId:       "RouterCheckOnlineForUniqId",       //检查某几个连接是否在线
	RouterForceOfflineForUserId:      "RouterForceOfflineForUserId",      //强制关闭某个连接
	RouterForceOfflineForUniqId:      "RouterForceOfflineForUniqId",      //强制关闭某个连接
	RouterForceOfflineGuestForUniqId: "RouterForceOfflineGuestForUniqId", //将某个没有session值的连接强制关闭
	RouterSingleCastForUserId:        "RouterSingleCastForUserId",        //单播给某个用户
	RouterSingleCastForUniqId:        "RouterSingleCastForUniqId",        //单播给某个uniqId
	RouterMulticastForUserId:         "RouterMulticastForUserId",         //组播给多个用户
	RouterMulticastForUniqId:         "RouterMulticastForUniqId",         //组播给多个uniqId
	RouterBroadcast:                  "RouterBroadcast",                  //广播给所有用户
	RouterTopicSubscribe:             "RouterTopicSubscribe",             //订阅
	RouterTopicUnsubscribe:           "RouterTopicUnsubscribe",           //取消订阅
	RouterTopicDelete:                "RouterTopicDelete",                //删除主题
	RouterTopicUniqIdCount:           "RouterTopicUniqIdCount",           //获取网关中的某几个主题的连接数
	RouterTopicUniqIdList:            "RouterTopicUniqIdList",            //获取网关中的某个主题包含的uniqId
	RouterTopicPublish:               "RouterTopicPublish",               //发布信息
}

type ClientRouter struct {
	Cmd  Cmd
	Data string
}

func ParseClientRouter(data []byte) *ClientRouter {
	ret := new(ClientRouter)
	if err := json.Unmarshal(data, ret); err != nil {
		log.Logger.Debug().Err(err).Msg("Parse ClientRouter failed")
		return nil
	}
	return ret
}
