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

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type topic struct{}

var Topic = topic{}

func init() {
	businessCmdCallback[protocol.RouterTopicCount] = Topic.RequestTopicCount
	businessCmdCallback[protocol.RouterTopicList] = Topic.RequestTopicList
	businessCmdCallback[protocol.RouterTopicUniqIdCount] = Topic.RequestTopicUniqIdCount
	businessCmdCallback[protocol.RouterTopicUniqIdList] = Topic.RequestTopicUniqIdList
	businessCmdCallback[protocol.RouterTopicCustomerIdList] = Topic.RequestTopicCustomerList
	businessCmdCallback[protocol.RouterTopicCustomerIdToUniqIdsList] = Topic.RequestTopicCustomerIdToUniqIdsList
	businessCmdCallback[protocol.RouterTopicCustomerIdCount] = Topic.RequestTopicCustomerCount
	businessCmdCallback[protocol.RouterTopicSubscribe] = Topic.RequestTopicSubscribe
	businessCmdCallback[protocol.RouterTopicUnsubscribe] = Topic.RequestTopicUnsubscribe
	businessCmdCallback[protocol.RouterTopicPublish] = Topic.RequestTopicPublish
	businessCmdCallback[protocol.RouterTopicPublishBulk] = Topic.RequestTopicPublishBulk
	businessCmdCallback[protocol.RouterTopicDelete] = Topic.RequestTopicDelete
}

// RequestTopicCount 获取网关中的主题数量
func (r topic) RequestTopicCount(tf *netsvrProtocol.Transfer, _ string) {
	resp := netBus.NetBus.TopicCount()
	msg := map[string]interface{}{"count": resp.Count()}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterTopicCount, map[string]interface{}{"code": 0, "message": "获取网关中的主题数量成功", "data": msg}))
}

// RequestTopicList 获取网关中的主题
func (topic) RequestTopicList(tf *netsvrProtocol.Transfer, _ string) {
	resp := netBus.NetBus.TopicList()
	msg := map[string]interface{}{"topics": resp.Data}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterTopicList, map[string]interface{}{"code": 0, "message": "获取网关中的主题成功", "data": msg}))
}

// TopicUniqIdCountParam 获取网关中的某几个主题的连接数
type TopicUniqIdCountParam struct {
	CountAll bool
	Topics   []string
}

// RequestTopicUniqIdCount 获取网关中的某几个主题的连接数
func (topic) RequestTopicUniqIdCount(tf *netsvrProtocol.Transfer, param string) {
	payload := TopicUniqIdCountParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicUniqIdCountParam failed")
		return
	}
	resp := netBus.NetBus.TopicUniqIdCount(payload.Topics, payload.CountAll)
	//将结果单播给客户端
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterTopicUniqIdCount, map[string]interface{}{"code": 0, "message": "获取网关中主题的连接数成功", "data": resp.Data}))
}

type TopicUniqIdListParam struct {
	Topics []string
}

// RequestTopicUniqIdList 获取网关中的某个主题包含的uniqId
func (topic) RequestTopicUniqIdList(tf *netsvrProtocol.Transfer, param string) {
	//解析客户端发来的数据
	payload := new(TopicUniqIdListParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicUniqIdListParam failed")
		return
	}
	resp := netBus.NetBus.TopicUniqIdList(payload.Topics)
	//将结果单播给客户端
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterTopicUniqIdList, map[string]interface{}{"code": 0, "message": "获取网关中的主题的uniqId成功", "data": resp.Data}))
}

// RequestTopicCustomerListParam 获取网关中某几个主题的customerId
type RequestTopicCustomerListParam struct {
	Topics []string `json:"topics"`
}

// RequestTopicCustomerList 获取网关中某几个主题的customerId
func (topic) RequestTopicCustomerList(tf *netsvrProtocol.Transfer, param string) {
	payload := &RequestTopicCustomerListParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse RequestTopicCustomerListParam failed")
		return
	}
	resp := netBus.NetBus.TopicCustomerIdList(payload.Topics)
	//将结果单播给客户端
	msg := map[string]interface{}{
		"list": resp.Data,
	}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterTopicCustomerIdList, map[string]interface{}{"code": 0, "message": "获取主题的customerId成功", "data": msg}))
}

// RequestTopicCustomerIdToUniqIdsListParam 获取网关中某几个主题的customerId以及对应的uniqId列表
type RequestTopicCustomerIdToUniqIdsListParam struct {
	Topics []string `json:"topics"`
}

// RequestTopicCustomerIdToUniqIdsList 获取网关中目标topic的customerId以及对应的uniqId列表
func (topic) RequestTopicCustomerIdToUniqIdsList(tf *netsvrProtocol.Transfer, param string) {
	payload := &RequestTopicCustomerIdToUniqIdsListParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse RequestTopicCustomerIdToUniqIdsListParam failed")
		return
	}
	resp := netBus.NetBus.TopicCustomerIdToUniqIdsList(payload.Topics)
	//将结果单播给客户端
	msg := map[string]interface{}{
		"list": resp.Data,
	}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterTopicCustomerIdToUniqIdsList, map[string]interface{}{"code": 0, "message": "获取主题的customerId以及对应的uniqId列表成功", "data": msg}))
}

// RequestTopicCustomerCountParam 获取网关中某几个主题的customerId
type RequestTopicCustomerCountParam struct {
	Topics []string `json:"topics"`
}

// RequestTopicCustomerCount 获取网关中某几个主题的customerId数量
func (topic) RequestTopicCustomerCount(tf *netsvrProtocol.Transfer, param string) {
	payload := &RequestTopicCustomerCountParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse RequestTopicCustomerCountParam failed")
		return
	}
	resp := netBus.NetBus.TopicCustomerIdCount(payload.Topics, false)
	//将结果单播给客户端
	msg := map[string]interface{}{
		"list": resp.Data,
	}
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterTopicCustomerIdCount, map[string]interface{}{"code": 0, "message": "获取主题的customerId数量成功", "data": msg}))
}

// TopicSubscribeParam 客户端发送的订阅信息
type TopicSubscribeParam struct {
	Topics []string
}

// RequestTopicSubscribe 处理客户的订阅请求
func (topic) RequestTopicSubscribe(tf *netsvrProtocol.Transfer, param string) {
	//解析客户端发来的数据
	payload := new(TopicSubscribeParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicSubscribeParam failed")
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	//提交订阅信息到网关
	netBus.NetBus.TopicSubscribe(tf.UniqId, payload.Topics, testUtils.NewResponse(protocol.RouterTopicSubscribe, map[string]interface{}{"code": 0, "message": "订阅成功", "data": nil}))
}

// TopicUnsubscribeParam 客户端发送的取消订阅信息
type TopicUnsubscribeParam struct {
	Topics []string
}

// RequestTopicUnsubscribe 处理客户的取消订阅请求
func (topic) RequestTopicUnsubscribe(tf *netsvrProtocol.Transfer, param string) {
	//解析客户端发来的数据
	payload := new(TopicUnsubscribeParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicUnsubscribeParam failed")
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	netBus.NetBus.TopicUnsubscribe(tf.UniqId, payload.Topics, testUtils.NewResponse(protocol.RouterTopicUnsubscribe, map[string]interface{}{"code": 0, "message": "取消订阅成功", "data": nil}))
}

// TopicPublishParam 客户端发送的发布信息
type TopicPublishParam struct {
	Message string
	Topics  []string
}

// RequestTopicPublish 处理客户的发布请求
func (topic) RequestTopicPublish(tf *netsvrProtocol.Transfer, param string) {
	//解析客户端发来的数据
	target := new(TopicPublishParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), target); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicPublishParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	msg := map[string]interface{}{"fromUser": fromUser, "message": target.Message}
	netBus.NetBus.TopicPublish(target.Topics, testUtils.NewResponse(protocol.RouterTopicPublish, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg}))
}

// TopicPublishBulkParam 客户端发送的批量发布信息
type TopicPublishBulkParam struct {
	Message []string
	Topics  []string
}

// RequestTopicPublishBulk 处理客户的批量发布请求
func (topic) RequestTopicPublishBulk(tf *netsvrProtocol.Transfer, param string) {
	//解析客户端发来的数据
	target := new(TopicPublishBulkParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), target); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicPublishBulkParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	bulkData := make([][]byte, 0, len(target.Message))
	for _, data := range target.Message {
		//这里message拼接上topic，方便界面上识别
		msg := map[string]interface{}{"fromUser": fromUser, "message": data}
		bulkData = append(bulkData, testUtils.NewResponse(protocol.RouterTopicPublishBulk, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg}))
	}
	netBus.NetBus.TopicPublishBulk(target.Topics, bulkData)
}

// TopicDeleteParam 客户端发送要删除的主题
type TopicDeleteParam struct {
	Topics []string
}

// RequestTopicDelete 处理客户的删除主题请求
func (topic) RequestTopicDelete(_ *netsvrProtocol.Transfer, param string) {
	//解析客户端发来的数据
	payload := new(TopicDeleteParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicDeleteParam failed")
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	netBus.NetBus.TopicDelete(payload.Topics, testUtils.NewResponse(protocol.RouterTopicDelete, map[string]interface{}{"code": 0, "message": "删除主题成功", "data": nil}))
}
