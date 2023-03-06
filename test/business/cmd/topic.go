package cmd

import (
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/log"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/test/business/connProcessor"
	"netsvr/test/business/userDb"
	"netsvr/test/protocol"
	businessUtils "netsvr/test/utils"
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
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.TopicCountResp failed")
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
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.TopicListResp failed")
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

// TopicUniqIdCountParam 获取网关中的某几个主题的连接数
type TopicUniqIdCountParam struct {
	CountAll bool
	Topics   []string
}

// RequestTopicUniqIdCount 获取网关中的某几个主题的连接数
func (topic) RequestTopicUniqIdCount(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := TopicUniqIdCountParam{}
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicUniqIdCountParam failed")
		return
	}
	req := internalProtocol.TopicUniqIdCountReq{}
	req.RouterCmd = int32(protocol.RouterTopicUniqIdCount)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Topics = payload.Topics
	req.CountAll = payload.CountAll
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicUniqIdCount
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicUniqIdCount 处理worker发送过来的主题的连接数
func (topic) ResponseTopicUniqIdCount(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.TopicUniqIdCountResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.TopicUniqIdCountResp failed")
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
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicUniqIdCount, map[string]interface{}{"code": 0, "message": "获取网关中主题的连接数成功", "data": payload.Items})
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// TopicUniqIdListParam 获取网关中的某个主题包含的uniqId
type TopicUniqIdListParam struct {
	Topic string
}

// RequestTopicUniqIdList 获取网关中的某个主题包含的uniqId
func (topic) RequestTopicUniqIdList(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicUniqIdListParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicUniqIdListParam failed")
		return
	}
	req := internalProtocol.TopicUniqIdListReq{}
	req.RouterCmd = int32(protocol.RouterTopicUniqIdList)
	req.CtxData = businessUtils.StrToReadOnlyBytes(tf.UniqId)
	req.Topic = payload.Topic
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicUniqIdList
	router.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// ResponseTopicUniqIdList 处理worker发送过来的主题包含的uniqId
func (topic) ResponseTopicUniqIdList(param []byte, processor *connProcessor.ConnProcessor) {
	payload := internalProtocol.TopicUniqIdListResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.TopicUniqIdListResp failed")
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
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicUniqIdList, map[string]interface{}{"code": 0, "message": "获取网关中的主题的uniqId成功", "data": payload.UniqIds})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_SingleCast
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
		log.Logger.Error().Err(err).Msg("Parse internalProtocol.InfoResp failed")
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

// TopicSubscribeParam 客户端发送的订阅信息
type TopicSubscribeParam struct {
	Topics []string
}

// RequestTopicSubscribe 处理客户的订阅请求
func (topic) RequestTopicSubscribe(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicSubscribeParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicSubscribeParam failed")
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

// TopicUnsubscribeParam 客户端发送的取消订阅信息
type TopicUnsubscribeParam struct {
	Topics []string
}

// RequestTopicUnsubscribe 处理客户的取消订阅请求
func (topic) RequestTopicUnsubscribe(tf *internalProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	payload := new(TopicUnsubscribeParam)
	if err := json.Unmarshal(businessUtils.StrToReadOnlyBytes(param), payload); err != nil {
		log.Logger.Error().Err(err).Msg("Parse TopicUnsubscribeParam failed")
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
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicUnsubscribe, map[string]interface{}{"code": 0, "message": "取消订阅成功", "data": nil})
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
	ret := &internalProtocol.TopicPublish{}
	ret.Topic = target.Topic
	ret.Data = businessUtils.NewResponse(protocol.RouterTopicPublish, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_TopicPublish
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
		log.Logger.Error().Err(err).Msg("Parse TopicDeleteParam failed")
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
