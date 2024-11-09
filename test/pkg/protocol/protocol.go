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
	Placeholder Cmd = iota
	RouterRespConnOpen
	RouterRespConnClose
	RouterMetrics
	RouterLimit
	RouterUniqIdList
	RouterUniqIdCount
	RouterCustomerIdList
	RouterCustomerIdCount
	RouterConnInfo
	RouterConnInfoByCustomerId
	RouterTopicCount
	RouterTopicList
	RouterSignIn
	RouterSignOut
	RouterSignInForForge
	RouterSignOutForForge
	RouterCheckOnline
	RouterForceOfflineByCustomerId
	RouterForceOffline
	RouterForceOfflineGuest
	RouterSingleCastByCustomerId
	RouterSingleCast
	RouterSingleCastBulk
	RouterSingleCastBulkByCustomerId
	RouterMulticast
	RouterMulticastByCustomerId
	RouterBroadcast
	RouterTopicSubscribe
	RouterTopicUnsubscribe
	RouterTopicDelete
	RouterTopicUniqIdCount
	RouterTopicUniqIdList
	RouterTopicCustomerIdList
	RouterTopicCustomerIdToUniqIdsList
	RouterTopicCustomerIdCount
	RouterTopicPublish
	RouterTopicPublishBulk
)

var CmdName = map[Cmd]string{
	RouterRespConnOpen:                 "RouterRespConnOpen",                 //响应连接打开信息
	RouterRespConnClose:                "RouterRespConnClose",                //响应连接关闭信息
	RouterMetrics:                      "RouterMetrics",                      //获取网关状态的统计信息
	RouterLimit:                        "RouterLimit",                        //更新限流配置、获取网关中的限流配置的真实情况
	RouterUniqIdList:                   "RouterUniqIdList",                   //获取网关中全部的uniqId
	RouterUniqIdCount:                  "RouterUniqIdCount",                  //获取网关中uniqId的数量
	RouterCustomerIdList:               "RouterCustomerIdList",               //获取网关所有customerId
	RouterCustomerIdCount:              "RouterCustomerIdCount",              //获取网关中的customerId数量
	RouterConnInfo:                     "RouterConnInfo",                     //获取我的连接信息
	RouterConnInfoByCustomerId:         "RouterConnInfoByCustomerId",         //获取customerId的连接信息
	RouterTopicCount:                   "RouterTopicCount",                   //获取网关中的主题
	RouterTopicList:                    "RouterTopicList",                    //获取网关中的主题数量
	RouterSignIn:                       "RouterSignIn",                       //登录
	RouterSignOut:                      "RouterSignOut",                      //退出登录
	RouterSignInForForge:               "RouterSignInForForge",               //伪造登录
	RouterSignOutForForge:              "RouterSignOutForForge",              //伪造退出登录
	RouterCheckOnline:                  "RouterCheckOnline",                  //检查某几个连接是否在线
	RouterForceOfflineByCustomerId:     "RouterForceOfflineByCustomerId",     //强制关闭某个连接
	RouterForceOffline:                 "RouterForceOffline",                 //强制关闭某个连接
	RouterForceOfflineGuest:            "RouterForceOfflineGuest",            //将某个没有session值的连接强制关闭
	RouterSingleCastByCustomerId:       "RouterSingleCastByCustomerId",       //单播给某个用户
	RouterSingleCastBulk:               "RouterSingleCastBulk",               //批量单播给某几个uniqId
	RouterSingleCastBulkByCustomerId:   "RouterSingleCastBulkByCustomerId",   //批量单播给某几个customerId
	RouterSingleCast:                   "RouterSingleCast",                   //单播给某个uniqId
	RouterMulticast:                    "RouterMulticast",                    //组播给多个uniqId
	RouterMulticastByCustomerId:        "RouterMulticastByCustomerId",        //组播给多个customerId
	RouterBroadcast:                    "RouterBroadcast",                    //广播给所有用户
	RouterTopicSubscribe:               "RouterTopicSubscribe",               //订阅
	RouterTopicUnsubscribe:             "RouterTopicUnsubscribe",             //取消订阅
	RouterTopicDelete:                  "RouterTopicDelete",                  //删除主题
	RouterTopicUniqIdCount:             "RouterTopicUniqIdCount",             //获取网关中的某几个主题的连接数
	RouterTopicUniqIdList:              "RouterTopicUniqIdList",              //获取网关中的某个主题包含的uniqId
	RouterTopicCustomerIdList:          "RouterTopicCustomerIdList",          //获取网关中某几个主题的customerId
	RouterTopicCustomerIdToUniqIdsList: "RouterTopicCustomerIdToUniqIdsList", //获取网关中目标topic的customerId以及对应的uniqId列表
	RouterTopicCustomerIdCount:         "RouterTopicCustomerIdCount",         //获取网关中某几个主题的customerId数量
	RouterTopicPublish:                 "RouterTopicPublish",                 //发布信息
	RouterTopicPublishBulk:             "RouterTopicPublishBulk",             //批量发布信息
}

type ClientRouter struct {
	Cmd  Cmd
	Data string
}
