/**
* Copyright 2022 buexplain@qq.com
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
	"google.golang.org/protobuf/proto"
	netsvrProtocol "netsvr/pkg/protocol"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type topic struct{}

var Topic = topic{}

func (r topic) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterTopicCount, r.RequestTopicCount)
	processor.RegisterWorkerCmd(protocol.RouterTopicCount, r.ResponseTopicCount)

	processor.RegisterBusinessCmd(protocol.RouterTopicList, r.RequestTopicList)
	processor.RegisterWorkerCmd(protocol.RouterTopicList, r.ResponseTopicList)

	processor.RegisterBusinessCmd(protocol.RouterTopicUniqIdCount, r.RequestTopicUniqIdCount)
	processor.RegisterWorkerCmd(protocol.RouterTopicUniqIdCount, r.ResponseTopicUniqIdCount)

	processor.RegisterBusinessCmd(protocol.RouterTopicUniqIdList, r.RequestTopicUniqIdList)
	processor.RegisterWorkerCmd(protocol.RouterTopicUniqIdList, r.ResponseTopicUniqIdList)

	processor.RegisterBusinessCmd(protocol.RouterTopicMyList, r.RequestTopicMyList)
	processor.RegisterWorkerCmd(protocol.RouterTopicMyList, r.ResponseTopicMyList)

	processor.RegisterBusinessCmd(protocol.RouterTopicSubscribe, r.RequestTopicSubscribe)
	processor.RegisterBusinessCmd(protocol.RouterTopicUnsubscribe, r.RequestTopicUnsubscribe)
	processor.RegisterBusinessCmd(protocol.RouterTopicPublish, r.RequestTopicPublish)
	processor.RegisterBusinessCmd(protocol.RouterTopicDelete, r.RequestTopicDelete)
}

// RequestTopicCount ??????????????????????????????
func (r topic) RequestTopicCount(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &netsvrProtocol.TopicCountReq{}
	req.RouterCmd = int32(protocol.RouterTopicCount)
	req.CtxData = testUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_TopicCount
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicCount ??????worker???????????????????????????????????????
func (r topic) ResponseTopicCount(param []byte, processor *connProcessor.ConnProcessor) {
	payload := &netsvrProtocol.TopicCountResp{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.TopicCountResp failed")
		return
	}
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = testUtils.BytesToReadOnlyString(payload.CtxData)
	msg := map[string]interface{}{"count": payload.Count}
	ret.Data = testUtils.NewResponse(protocol.RouterTopicCount, map[string]interface{}{"code": 0, "message": "????????????????????????????????????", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// RequestTopicList ????????????????????????
func (topic) RequestTopicList(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &netsvrProtocol.TopicListReq{}
	req.RouterCmd = int32(protocol.RouterTopicList)
	req.CtxData = testUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_TopicList
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicList ??????worker?????????????????????????????????
func (topic) ResponseTopicList(param []byte, processor *connProcessor.ConnProcessor) {
	payload := &netsvrProtocol.TopicListResp{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.TopicListResp failed")
		return
	}
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = testUtils.BytesToReadOnlyString(payload.CtxData)
	msg := map[string]interface{}{"topics": payload.Topics}
	ret.Data = testUtils.NewResponse(protocol.RouterTopicList, map[string]interface{}{"code": 0, "message": "??????????????????????????????", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicUniqIdCountParam ?????????????????????????????????????????????
type TopicUniqIdCountParam struct {
	CountAll bool
	Topics   []string
}

// RequestTopicUniqIdCount ?????????????????????????????????????????????
func (topic) RequestTopicUniqIdCount(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := TopicUniqIdCountParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicUniqIdCountParam failed")
		return
	}
	req := netsvrProtocol.TopicUniqIdCountReq{}
	req.RouterCmd = int32(protocol.RouterTopicUniqIdCount)
	req.CtxData = testUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Topics = payload.Topics
	req.CountAll = payload.CountAll
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_TopicUniqIdCount
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicUniqIdCount ??????worker?????????????????????????????????
func (topic) ResponseTopicUniqIdCount(param []byte, processor *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.TopicUniqIdCountResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.TopicUniqIdCountResp failed")
		return
	}
	//?????????????????????????????????uniqId
	uniqId := testUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//???????????????????????????
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = testUtils.NewResponse(protocol.RouterTopicUniqIdCount, map[string]interface{}{"code": 0, "message": "???????????????????????????????????????", "data": payload.Items})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicUniqIdListParam ???????????????????????????????????????uniqId
type TopicUniqIdListParam struct {
	Topic string
}

// RequestTopicUniqIdList ???????????????????????????????????????uniqId
func (topic) RequestTopicUniqIdList(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//??????????????????????????????
	payload := new(TopicUniqIdListParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicUniqIdListParam failed")
		return
	}
	req := netsvrProtocol.TopicUniqIdListReq{}
	req.RouterCmd = int32(protocol.RouterTopicUniqIdList)
	req.CtxData = testUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Topic = payload.Topic
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_TopicUniqIdList
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicUniqIdList ??????worker??????????????????????????????uniqId
func (topic) ResponseTopicUniqIdList(param []byte, processor *connProcessor.ConnProcessor) {
	payload := netsvrProtocol.TopicUniqIdListResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.TopicUniqIdListResp failed")
		return
	}
	//?????????????????????????????????uniqId
	uniqId := testUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//???????????????????????????
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = testUtils.NewResponse(protocol.RouterTopicUniqIdList, map[string]interface{}{"code": 0, "message": "???????????????????????????uniqId??????", "data": payload.UniqIds})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// RequestTopicMyList ?????????????????????????????????
func (topic) RequestTopicMyList(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	//??????????????????info????????????????????????????????????
	//????????????????????????????????????????????????????????????????????????
	req := &netsvrProtocol.InfoReq{}
	req.RouterCmd = int32(protocol.RouterTopicMyList)
	req.UniqId = tf.UniqId
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_Info
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicMyList ??????worker??????????????????????????????session??????
func (topic) ResponseTopicMyList(param []byte, processor *connProcessor.ConnProcessor) {
	payload := &netsvrProtocol.InfoResp{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.InfoResp failed")
		return
	}
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = payload.UniqId
	msg := map[string]interface{}{"topics": payload.Topics}
	ret.Data = testUtils.NewResponse(protocol.RouterTopicMyList, map[string]interface{}{"code": 0, "message": "??????????????????????????????", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicSubscribeParam ??????????????????????????????
type TopicSubscribeParam struct {
	Topics []string
}

// RequestTopicSubscribe ???????????????????????????
func (topic) RequestTopicSubscribe(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//??????????????????????????????
	payload := new(TopicSubscribeParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicSubscribeParam failed")
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	//???????????????????????????
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_TopicSubscribe
	ret := &netsvrProtocol.TopicSubscribe{}
	ret.UniqId = tf.UniqId
	ret.Topics = payload.Topics
	ret.Data = testUtils.NewResponse(protocol.RouterTopicSubscribe, map[string]interface{}{"code": 0, "message": "????????????", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicUnsubscribeParam ????????????????????????????????????
type TopicUnsubscribeParam struct {
	Topics []string
}

// RequestTopicUnsubscribe ?????????????????????????????????
func (topic) RequestTopicUnsubscribe(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//??????????????????????????????
	payload := new(TopicUnsubscribeParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicUnsubscribeParam failed")
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_TopicUnsubscribe
	ret := &netsvrProtocol.TopicUnsubscribe{}
	ret.UniqId = tf.UniqId
	ret.Topics = payload.Topics
	ret.Data = testUtils.NewResponse(protocol.RouterTopicUnsubscribe, map[string]interface{}{"code": 0, "message": "??????????????????", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicPublishParam ??????????????????????????????
type TopicPublishParam struct {
	Message string
	Topic   string
}

// RequestTopicPublish ???????????????????????????
func (topic) RequestTopicPublish(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//??????????????????????????????
	target := new(TopicPublishParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), target); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicPublishParam failed")
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
	ret.Topic = target.Topic
	ret.Data = testUtils.NewResponse(protocol.RouterTopicPublish, map[string]interface{}{"code": 0, "message": "??????????????????", "data": msg})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_TopicPublish
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicDeleteParam ?????????????????????????????????
type TopicDeleteParam struct {
	Topics []string
}

// RequestTopicDelete ?????????????????????????????????
func (topic) RequestTopicDelete(_ *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//??????????????????????????????
	payload := new(TopicDeleteParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicDeleteParam failed")
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_TopicDelete
	ret := &netsvrProtocol.TopicDelete{}
	ret.Topics = payload.Topics
	ret.Data = testUtils.NewResponse(protocol.RouterTopicDelete, map[string]interface{}{"code": 0, "message": "??????????????????", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
