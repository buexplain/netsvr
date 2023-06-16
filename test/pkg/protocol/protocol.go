/**
* Copyright 2023 buexplain@qq.com
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package protocol

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
	RouterSingleCastBulkForUniqId
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
	RouterRespConnOpen:               "RouterRespConnOpen",               //响应连接打开信息
	RouterRespConnClose:              "RouterRespConnClose",              //响应连接关闭信息
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
	RouterSingleCastBulkForUniqId:    "RouterSingleCastBulkForUniqId",    //批量单播给某几个uniqId
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
