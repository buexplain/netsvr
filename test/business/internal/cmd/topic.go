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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v3/netsvr"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
	"netsvr/test/pkg/utils/netSvrPool"
)

type topic struct{}

var Topic = topic{}

func (r topic) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterTopicCount, r.RequestTopicCount)
	processor.RegisterBusinessCmd(protocol.RouterTopicList, r.RequestTopicList)
	processor.RegisterBusinessCmd(protocol.RouterTopicUniqIdCount, r.RequestTopicUniqIdCount)
	processor.RegisterBusinessCmd(protocol.RouterTopicUniqIdList, r.RequestTopicUniqIdList)
	processor.RegisterBusinessCmd(protocol.RouterTopicCustomerIdList, r.RequestTopicCustomerList)
	processor.RegisterBusinessCmd(protocol.RouterTopicCustomerIdCount, r.RequestTopicCustomerCount)
	processor.RegisterBusinessCmd(protocol.RouterTopicSubscribe, r.RequestTopicSubscribe)
	processor.RegisterBusinessCmd(protocol.RouterTopicUnsubscribe, r.RequestTopicUnsubscribe)
	processor.RegisterBusinessCmd(protocol.RouterTopicPublish, r.RequestTopicPublish)
	processor.RegisterBusinessCmd(protocol.RouterTopicPublishBulk, r.RequestTopicPublishBulk)
	processor.RegisterBusinessCmd(protocol.RouterTopicDelete, r.RequestTopicDelete)
}

// RequestTopicCount 获取网关中的主题数量
func (r topic) RequestTopicCount(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	resp := &netsvrProtocol.TopicCountResp{}
	netSvrPool.Request(nil, netsvrProtocol.Cmd_TopicCount, resp)
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{"count": resp.Count}
	ret.Data = testUtils.NewResponse(protocol.RouterTopicCount, map[string]interface{}{"code": 0, "message": "获取网关中的主题数量成功", "data": msg})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}

// RequestTopicList 获取网关中的主题
func (topic) RequestTopicList(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	resp := &netsvrProtocol.TopicListResp{}
	netSvrPool.Request(nil, netsvrProtocol.Cmd_TopicList, resp)
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{"topics": resp.Topics}
	ret.Data = testUtils.NewResponse(protocol.RouterTopicList, map[string]interface{}{"code": 0, "message": "获取网关中的主题成功", "data": msg})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}

// TopicUniqIdCountParam 获取网关中的某几个主题的连接数
type TopicUniqIdCountParam struct {
	CountAll bool
	Topics   []string
}

// RequestTopicUniqIdCount 获取网关中的某几个主题的连接数
func (topic) RequestTopicUniqIdCount(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := TopicUniqIdCountParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicUniqIdCountParam failed")
		return
	}
	req := &netsvrProtocol.TopicUniqIdCountReq{}
	req.Topics = payload.Topics
	req.CountAll = payload.CountAll
	resp := &netsvrProtocol.TopicUniqIdCountResp{}
	netSvrPool.Request(req, netsvrProtocol.Cmd_TopicUniqIdCount, resp)
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	ret.Data = testUtils.NewResponse(protocol.RouterTopicUniqIdCount, map[string]interface{}{"code": 0, "message": "获取网关中主题的连接数成功", "data": resp.Items})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}

type TopicUniqIdListParam struct {
	Topic string
}

// RequestTopicUniqIdList 获取网关中的某个主题包含的uniqId
func (topic) RequestTopicUniqIdList(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicUniqIdListParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicUniqIdListParam failed")
		return
	}
	req := &netsvrProtocol.TopicUniqIdListReq{}
	req.Topics = []string{payload.Topic}
	resp := &netsvrProtocol.TopicUniqIdListResp{}
	netSvrPool.Request(req, netsvrProtocol.Cmd_TopicUniqIdList, resp)
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	ret.Data = testUtils.NewResponse(protocol.RouterTopicUniqIdList, map[string]interface{}{"code": 0, "message": "获取网关中的主题的uniqId成功", "data": resp.Items})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}

// RequestTopicCustomerListParam 获取网关中某几个主题的customerId
type RequestTopicCustomerListParam struct {
	Topics []string `json:"topics"`
}

// RequestTopicCustomerList 获取网关中某几个主题的customerId
func (topic) RequestTopicCustomerList(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := &RequestTopicCustomerListParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse RequestTopicCustomerListParam failed")
		return
	}
	req := &netsvrProtocol.TopicCustomerIdListReq{}
	req.Topics = payload.Topics
	resp := &netsvrProtocol.TopicCustomerIdListResp{}
	netSvrPool.Request(req, netsvrProtocol.Cmd_TopicCustomerIdList, resp)
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{
		"list": resp.Items,
	}
	ret.Data = testUtils.NewResponse(protocol.RouterTopicCustomerIdList, map[string]interface{}{"code": 0, "message": "获取主题的customerId成功", "data": msg})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}

// RequestTopicCustomerCountParam 获取网关中某几个主题的customerId
type RequestTopicCustomerCountParam struct {
	Topics []string `json:"topics"`
}

// RequestTopicCustomerCount 获取网关中某几个主题的customerId数量
func (topic) RequestTopicCustomerCount(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := &RequestTopicCustomerCountParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse RequestTopicCustomerCountParam failed")
		return
	}
	req := &netsvrProtocol.TopicCustomerIdCountReq{}
	req.Topics = payload.Topics
	resp := &netsvrProtocol.TopicCustomerIdCountResp{}
	netSvrPool.Request(req, netsvrProtocol.Cmd_TopicCustomerIdCount, resp)
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{
		"list": resp.Items,
	}
	ret.Data = testUtils.NewResponse(protocol.RouterTopicCustomerIdCount, map[string]interface{}{"code": 0, "message": "获取主题的customerId数量成功", "data": msg})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}

// TopicSubscribeParam 客户端发送的订阅信息
type TopicSubscribeParam struct {
	Topics []string
}

// RequestTopicSubscribe 处理客户的订阅请求
func (topic) RequestTopicSubscribe(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
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
	ret := &netsvrProtocol.TopicSubscribe{}
	ret.UniqId = tf.UniqId
	ret.Topics = payload.Topics
	ret.Data = testUtils.NewResponse(protocol.RouterTopicSubscribe, map[string]interface{}{"code": 0, "message": "订阅成功", "data": nil})
	processor.Send(ret, netsvrProtocol.Cmd_TopicSubscribe)
}

// TopicUnsubscribeParam 客户端发送的取消订阅信息
type TopicUnsubscribeParam struct {
	Topics []string
}

// RequestTopicUnsubscribe 处理客户的取消订阅请求
func (topic) RequestTopicUnsubscribe(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicUnsubscribeParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicUnsubscribeParam failed")
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	ret := &netsvrProtocol.TopicUnsubscribe{}
	ret.UniqId = tf.UniqId
	ret.Topics = payload.Topics
	ret.Data = testUtils.NewResponse(protocol.RouterTopicUnsubscribe, map[string]interface{}{"code": 0, "message": "取消订阅成功", "data": nil})
	processor.Send(ret, netsvrProtocol.Cmd_TopicUnsubscribe)
}

// TopicPublishParam 客户端发送的发布信息
type TopicPublishParam struct {
	Message string
	Topics  []string
}

// RequestTopicPublish 处理客户的发布请求
func (topic) RequestTopicPublish(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
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
	ret := &netsvrProtocol.TopicPublish{}
	ret.Topics = target.Topics
	ret.Data = testUtils.NewResponse(protocol.RouterTopicPublish, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	processor.Send(ret, netsvrProtocol.Cmd_TopicPublish)
}

// TopicPublishBulkParam 客户端发送的批量发布信息
type TopicPublishBulkParam struct {
	Message []string
	Topics  []string
}

// RequestTopicPublishBulk 处理客户的批量发布请求
func (topic) RequestTopicPublishBulk(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
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
	ret := &netsvrProtocol.TopicPublishBulk{Topics: make([]string, 0, len(target.Topics)), Data: make([][]byte, 0, len(target.Topics))}
	for _, data := range target.Message {
		//这里message拼接上topic，方便界面上识别
		msg := map[string]interface{}{"fromUser": fromUser, "message": data}
		ret.Data = append(ret.Data, testUtils.NewResponse(protocol.RouterTopicPublishBulk, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg}))
	}
	ret.Topics = target.Topics
	processor.Send(ret, netsvrProtocol.Cmd_TopicPublishBulk)
}

// TopicDeleteParam 客户端发送要删除的主题
type TopicDeleteParam struct {
	Topics []string
}

// RequestTopicDelete 处理客户的删除主题请求
func (topic) RequestTopicDelete(_ *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicDeleteParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse TopicDeleteParam failed")
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	ret := &netsvrProtocol.TopicDelete{}
	ret.Topics = payload.Topics
	ret.Data = testUtils.NewResponse(protocol.RouterTopicDelete, map[string]interface{}{"code": 0, "message": "删除主题成功", "data": nil})
	processor.Send(ret, netsvrProtocol.Cmd_TopicDelete)
}
