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
)

type topic struct{}

var Topic = topic{}

func (r topic) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterTopicCount, r.RequestTopicCount)
	processor.RegisterWorkerCmd(protocol.RouterTopicCount, r.ResponseTopicCount)
	processor.RegisterBusinessCmd(protocol.RouterTopicDelete, r.RequestTopicDelete)
	processor.RegisterBusinessCmd(protocol.RouterTopicList, r.RequestTopicList)
	processor.RegisterWorkerCmd(protocol.RouterTopicList, r.ResponseTopicList)
	processor.RegisterBusinessCmd(protocol.RouterTopicMyList, r.RequestTopicMyList)
	processor.RegisterWorkerCmd(protocol.RouterTopicMyList, r.ResponseTopicMyList)
	processor.RegisterBusinessCmd(protocol.RouterTopicPublish, r.RequestTopicPublish)
	processor.RegisterBusinessCmd(protocol.RouterTopicSubscribe, r.RequestTopicSubscribe)
	processor.RegisterBusinessCmd(protocol.RouterTopicsUniqIdCount, r.RequestTopicsUniqIdCount)
	processor.RegisterWorkerCmd(protocol.RouterTopicsUniqIdCount, r.ResponseTopicsUniqIdCount)
	processor.RegisterWorkerCmd(protocol.RouterTopicsUniqIdCount, r.ResponseTopicsUniqIdCount)
	processor.RegisterBusinessCmd(protocol.RouterTopicUniqIds, r.RequestTopicUniqIds)
	processor.RegisterWorkerCmd(protocol.RouterTopicUniqIds, r.ResponseTopicUniqIds)
	processor.RegisterBusinessCmd(protocol.RouterTopicUnsubscribe, r.RequestTopicUnsubscribe)
}

// RequestTopicCount 获取网关中的主题数量
func (r topic) RequestTopicCount(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.TopicCountReq{}
	req.RouterCmd = int32(protocol.RouterTopicCount)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicCount
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicCount 处理worker发送过来的网关中的主题数量
func (r topic) ResponseTopicCount(param []byte, processor *connProcessor.ConnProcessor) {
	payload := &internalProtocol.TopicCountResp{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TopicCountResp error: %v", err)
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = businessUtils.BytesToReadOnlyString(payload.CtxData)
	msg := map[string]interface{}{"count": payload.Count}
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicCount, map[string]interface{}{"code": 0, "message": "获取网关中的主题数量成功", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicDeleteParam 客户端发送要删除的主题
type TopicDeleteParam struct {
	Topics []string
}

// RequestTopicDelete 处理客户的删除主题请求
func (topic) RequestTopicDelete(_ *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicDeleteParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse TopicDeleteParam error: %v", err)
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicDelete
	ret := &internalProtocol.TopicDelete{}
	ret.Topics = payload.Topics
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicDelete, map[string]interface{}{"code": 0, "message": "删除主题成功", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// RequestTopicList 获取网关中的主题
func (topic) RequestTopicList(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &internalProtocol.TopicListReq{}
	req.RouterCmd = int32(protocol.RouterTopicList)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicList
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicList 处理worker发送过来的网关中的主题
func (topic) ResponseTopicList(param []byte, processor *connProcessor.ConnProcessor) {
	payload := &internalProtocol.TopicListResp{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TopicListResp error: %v", err)
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = businessUtils.BytesToReadOnlyString(payload.CtxData)
	msg := map[string]interface{}{"topics": payload.Topics}
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicList, map[string]interface{}{"code": 0, "message": "获取网关中的主题成功", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// RequestTopicMyList 获取我已订阅的主题列表
func (topic) RequestTopicMyList(tf *internalProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	//获取网关中的info信息，从中提取出主题列表
	//正常的业务逻辑应该是查询数据库获取用户订阅的主题
	req := &internalProtocol.InfoReq{}
	req.RouterCmd = int32(protocol.RouterTopicMyList)
	req.UniqId = tf.UniqId
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Info
	router.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicMyList 处理worker发送过来的某个用户的session信息
func (topic) ResponseTopicMyList(param []byte, processor *connProcessor.ConnProcessor) {
	payload := &internalProtocol.InfoResp{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.InfoResp error: %v", err)
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = payload.UniqId
	msg := map[string]interface{}{"topics": payload.Topics}
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicMyList, map[string]interface{}{"code": 0, "message": "获取已订阅的主题成功", "data": msg})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicPublishParam 客户端发送的发布信息
type TopicPublishParam struct {
	Message string
	Topic   string
}

// RequestTopicPublish 处理客户的发布请求
func (topic) RequestTopicPublish(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(TopicPublishParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), target); err != nil {
		logging.Error("Parse TopicPublishParam error: %v", err)
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
	ret := &internalProtocol.TopicPublish{}
	ret.Topic = target.Topic
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicPublish, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicPublish
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicSubscribeParam 客户端发送的订阅信息
type TopicSubscribeParam struct {
	Topics []string
}

// RequestTopicSubscribe 处理客户的订阅请求
func (topic) RequestTopicSubscribe(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicSubscribeParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse TopicSubscribeParam error: %v", err)
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	//提交订阅信息到网关
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicSubscribe
	ret := &internalProtocol.TopicSubscribe{}
	ret.UniqId = tf.UniqId
	ret.Topics = payload.Topics
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicSubscribe, map[string]interface{}{"code": 0, "message": "订阅成功", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicsUniqIdCountParam 获取网关中的某几个主题的连接数
type TopicsUniqIdCountParam struct {
	CountAll bool
	Topics   []string
}

// RequestTopicsUniqIdCount 获取网关中的某几个主题的连接数
func (topic) RequestTopicsUniqIdCount(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := TopicsUniqIdCountParam{}
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		logging.Error("Parse TopicsUniqIdCountParam error: %v", err)
		return
	}
	req := internalProtocol.TopicsUniqIdCountReq{}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.RouterCmd = int32(protocol.RouterTopicsUniqIdCount)
	//将uniqId存储到请求上下文中去，当网关返回数据的时候用的上
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Topics = payload.Topics
	req.CountAll = payload.CountAll
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicsUniqIdCount
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicsUniqIdCount 处理worker发送过来的主题的连接数
func (topic) ResponseTopicsUniqIdCount(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.TopicsUniqIdCountResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TopicsUniqIdCountResp error: %v", err)
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := businessUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicsUniqIdCount, map[string]interface{}{"code": 0, "message": "获取网关中主题的连接数成功", "data": payload.Items})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicUniqIdsParam 获取网关中的某个主题包含的uniqId
type TopicUniqIdsParam struct {
	Topic string
}

// RequestTopicUniqIds 获取网关中的某个主题包含的uniqId
func (topic) RequestTopicUniqIds(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicUniqIdsParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse TopicUniqIdsParam error: %v", err)
		return
	}
	req := internalProtocol.TopicUniqIdsReq{}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.RouterCmd = int32(protocol.RouterTopicUniqIds)
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Topic = payload.Topic
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicUniqIds
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicUniqIds 处理worker发送过来的主题的连接session id
func (topic) ResponseTopicUniqIds(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.TopicUniqIdsResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal internalProtocol.TopicUniqIdsResp error: %v", err)
		return
	}
	//解析请求上下文中存储的uniqId
	uniqId := businessUtils.BytesToReadOnlyString(payload.CtxData)
	if uniqId == "" {
		return
	}
	//将结果单播给客户端
	ret := &internalProtocol.SingleCast{}
	ret.UniqId = uniqId
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicUniqIds, map[string]interface{}{"code": 0, "message": "获取网关中的主题的uniqId成功", "data": payload.UniqIds})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicUnsubscribeParam 客户端发送的取消订阅信息
type TopicUnsubscribeParam struct {
	Topics []string
}

// RequestTopicUnsubscribe 处理客户的取消订阅请求
func (topic) RequestTopicUnsubscribe(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicUnsubscribeParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), payload); err != nil {
		logging.Error("Parse TopicUnsubscribeParam error: %v", err)
		return
	}
	if len(payload.Topics) == 0 {
		return
	}
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicUnsubscribe
	ret := &internalProtocol.TopicUnsubscribe{}
	ret.UniqId = tf.UniqId
	ret.Topics = payload.Topics
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicSubscribe, map[string]interface{}{"code": 0, "message": "取消订阅成功", "data": nil})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
