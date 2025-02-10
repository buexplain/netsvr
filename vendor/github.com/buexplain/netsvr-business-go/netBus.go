/**
* Copyright 2024 buexplain@qq.com
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

package netsvrBusiness

import (
	"encoding/binary"
	"fmt"
	"github.com/buexplain/netsvr-business-go/resp"
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"google.golang.org/protobuf/proto"
)

type NetBus struct {
	mainSocketManager    *MainSocketManager
	taskSocketPoolManger *TaskSocketPoolManger
}

func NewNetBus(mainSocketManager *MainSocketManager, taskSocketPoolManger *TaskSocketPoolManger) *NetBus {
	if taskSocketPoolManger == nil {
		panic("taskSocketPoolManger is nil")
	}
	return &NetBus{
		mainSocketManager:    mainSocketManager,
		taskSocketPoolManger: taskSocketPoolManger,
	}
}

// ConnInfoUpdate 更新客户在网关存储的信息
func (n *NetBus) ConnInfoUpdate(connInfoUpdate *netsvrProtocol.ConnInfoUpdate) {
	message := n.pack(netsvrProtocol.Cmd_ConnInfoUpdate, connInfoUpdate)
	n.sendToSocketByUniqId(connInfoUpdate.GetUniqId(), message)
}

// ConnInfoDelete 删除目标uniqId在网关中存储的信息
func (n *NetBus) ConnInfoDelete(connInfoDelete *netsvrProtocol.ConnInfoDelete) {
	message := n.pack(netsvrProtocol.Cmd_ConnInfoDelete, connInfoDelete)
	n.sendToSocketByUniqId(connInfoDelete.GetUniqId(), message)
}

// Broadcast 广播
func (n *NetBus) Broadcast(data []byte) {
	broadcast := netsvrProtocol.Broadcast{
		Data: data,
	}
	message := n.pack(netsvrProtocol.Cmd_Broadcast, &broadcast)
	n.sendToSockets(message)
}

// Multicast 按uniqId组播
func (n *NetBus) Multicast(uniqIds []string, data []byte) {
	if n.isSinglePoint() || len(uniqIds) == 1 {
		multicast := netsvrProtocol.Multicast{
			UniqIds: uniqIds,
			Data:    data,
		}
		message := n.pack(netsvrProtocol.Cmd_Multicast, &multicast)
		n.sendToSocketByUniqId(uniqIds[0], message)
		return
	}
	group := n.getUniqIdsGroupByWorkerAddrAsHex(uniqIds)
	for workerAddrAsHex, currentUniqIds := range group {
		multicast := netsvrProtocol.Multicast{
			UniqIds: currentUniqIds,
			Data:    data,
		}
		message := n.pack(netsvrProtocol.Cmd_Multicast, &multicast)
		n.sendToSocketByWorkerAddrAsHex(workerAddrAsHex, message)
	}
}

// MulticastByCustomerId 按customerId组播
func (n *NetBus) MulticastByCustomerId(customerIds []string, data []byte) {
	multicastByCustomerId := netsvrProtocol.MulticastByCustomerId{}
	multicastByCustomerId.CustomerIds = customerIds
	multicastByCustomerId.Data = data
	message := n.pack(netsvrProtocol.Cmd_MulticastByCustomerId, &multicastByCustomerId)
	//因为不知道客户id在哪个网关，所以给所有网关发送
	n.sendToSockets(message)
}

// SingleCast 按uniqId单播
func (n *NetBus) SingleCast(uniqId string, data []byte) {
	singleCast := netsvrProtocol.SingleCast{
		UniqId: uniqId,
		Data:   data,
	}
	message := n.pack(netsvrProtocol.Cmd_SingleCast, &singleCast)
	n.sendToSocketByUniqId(uniqId, message)
}

// SingleCastByCustomerId 按customerId单播
func (n *NetBus) SingleCastByCustomerId(customerId string, data []byte) {
	singleCastByCustomerId := netsvrProtocol.SingleCastByCustomerId{}
	singleCastByCustomerId.CustomerId = customerId
	singleCastByCustomerId.Data = data
	message := n.pack(netsvrProtocol.Cmd_SingleCastByCustomerId, &singleCastByCustomerId)
	n.sendToSockets(message)
}

// SingleCastBulk 按uniqId批量单播，一次性给多个用户发送不同的消息，或给一个用户发送多条消息
func (n *NetBus) SingleCastBulk(uniqIds []string, data [][]byte) {
	//网关是单机部署或者是只给一个用户发消息，则直接构造批量单播对象发送
	if n.isSinglePoint() || len(uniqIds) == 1 {
		singleCastBulk := netsvrProtocol.SingleCastBulk{}
		singleCastBulk.Data = data
		singleCastBulk.UniqIds = uniqIds
		message := n.pack(netsvrProtocol.Cmd_SingleCastBulk, &singleCastBulk)
		n.sendToSocketByUniqId(uniqIds[0], message)
		return
	}
	//网关是多机器部署，或者是发个多个uniqId，需要迭代每一个uniqId，并根据所在网关进行分组，然后再迭代每一个组，将数据发送到对应网关
	type bulk struct {
		uniqIds []string
		data    [][]byte
	}
	bulks := make(map[string]*bulk)
	for index, uniqId := range uniqIds {
		workerAddrAsHex := UniqIdConvertToWorkerAddrAsHex(uniqId)
		if b, ok := bulks[workerAddrAsHex]; ok {
			b.uniqIds = append(b.uniqIds, uniqId)
			b.data = append(b.data, data[index])
		} else {
			b := &bulk{
				uniqIds: []string{uniqId},
				data:    [][]byte{data[index]},
			}
			bulks[workerAddrAsHex] = b
		}
	}
	//分组完毕，循环发送到各个网关
	for workerAddrAsHex, b := range bulks {
		singleCastBulk := netsvrProtocol.SingleCastBulk{}
		singleCastBulk.Data = b.data
		singleCastBulk.UniqIds = b.uniqIds
		n.sendToSocketByWorkerAddrAsHex(workerAddrAsHex, n.pack(netsvrProtocol.Cmd_SingleCastBulk, &singleCastBulk))
	}
}

// SingleCastBulkByCustomerId 按customerId批量单播，一次性给多个用户发送不同的消息，或给一个用户发送多条消息
func (n *NetBus) SingleCastBulkByCustomerId(customerIds []string, data [][]byte) {
	singleCastBulkByCustomerId := netsvrProtocol.SingleCastBulkByCustomerId{}
	singleCastBulkByCustomerId.CustomerIds = customerIds
	singleCastBulkByCustomerId.Data = data
	message := n.pack(netsvrProtocol.Cmd_SingleCastBulkByCustomerId, &singleCastBulkByCustomerId)
	n.sendToSockets(message)
}

// TopicSubscribe 订阅若干个主题
func (n *NetBus) TopicSubscribe(uniqId string, topics []string, data []byte) {
	topicSubscribe := netsvrProtocol.TopicSubscribe{}
	topicSubscribe.UniqId = uniqId
	topicSubscribe.Topics = topics
	topicSubscribe.Data = data
	message := n.pack(netsvrProtocol.Cmd_TopicSubscribe, &topicSubscribe)
	n.sendToSocketByUniqId(uniqId, message)
}

// TopicUnsubscribe 取消若干个已订阅的主题
func (n *NetBus) TopicUnsubscribe(uniqId string, topics []string, data []byte) {
	topicUnsubscribe := netsvrProtocol.TopicUnsubscribe{}
	topicUnsubscribe.UniqId = uniqId
	topicUnsubscribe.Topics = topics
	topicUnsubscribe.Data = data
	message := n.pack(netsvrProtocol.Cmd_TopicUnsubscribe, &topicUnsubscribe)
	n.sendToSocketByUniqId(uniqId, message)
}

// TopicDelete 删除若干个主题
func (n *NetBus) TopicDelete(topics []string, data []byte) {
	topicDelete := netsvrProtocol.TopicDelete{}
	topicDelete.Topics = topics
	topicDelete.Data = data
	message := n.pack(netsvrProtocol.Cmd_TopicDelete, &topicDelete)
	n.sendToSockets(message)
}

// TopicPublish 发布若干个主题
func (n *NetBus) TopicPublish(topics []string, data []byte) {
	topicPublish := netsvrProtocol.TopicPublish{}
	topicPublish.Topics = topics
	topicPublish.Data = data
	message := n.pack(netsvrProtocol.Cmd_TopicPublish, &topicPublish)
	n.sendToSockets(message)
}

// TopicPublishBulk 批量发布，一次性给多个主题发送不同的消息，或给一个主题发送多条消息
func (n *NetBus) TopicPublishBulk(topics []string, data [][]byte) {
	topicPublishBulk := netsvrProtocol.TopicPublishBulk{}
	topicPublishBulk.Data = data
	topicPublishBulk.Topics = topics
	message := n.pack(netsvrProtocol.Cmd_TopicPublishBulk, &topicPublishBulk)
	n.sendToSockets(message)
}

// ForceOffline 强制关闭某几个连接
func (n *NetBus) ForceOffline(uniqIds []string, data []byte) {
	if n.isSinglePoint() || len(uniqIds) == 1 {
		forceOffline := netsvrProtocol.ForceOffline{}
		forceOffline.UniqIds = uniqIds
		forceOffline.Data = data
		n.sendToSocketByUniqId(uniqIds[0], n.pack(netsvrProtocol.Cmd_ForceOffline, &forceOffline))
		return
	}
	group := n.getUniqIdsGroupByWorkerAddrAsHex(uniqIds)
	for workerAddrAsHex, currentUniqIds := range group {
		forceOffline := netsvrProtocol.ForceOffline{}
		forceOffline.UniqIds = currentUniqIds
		forceOffline.Data = data
		n.sendToSocketByWorkerAddrAsHex(workerAddrAsHex, n.pack(netsvrProtocol.Cmd_ForceOffline, &forceOffline))
	}
}

// ForceOfflineByCustomerId 强制关闭某几个customerId
func (n *NetBus) ForceOfflineByCustomerId(customerIds []string, data []byte) {
	forceOfflineByCustomerId := netsvrProtocol.ForceOfflineByCustomerId{}
	forceOfflineByCustomerId.CustomerIds = customerIds
	forceOfflineByCustomerId.Data = data
	message := n.pack(netsvrProtocol.Cmd_ForceOfflineByCustomerId, &forceOfflineByCustomerId)
	//因为不知道客户id在哪个网关，所以给所有网关发送
	n.sendToSockets(message)
}

// ForceOfflineGuest 强制关闭某几个空session值的连接
func (n *NetBus) ForceOfflineGuest(uniqIds []string, data []byte, delay int32) {
	if n.isSinglePoint() || len(uniqIds) == 1 {
		forceOfflineGuest := netsvrProtocol.ForceOfflineGuest{}
		forceOfflineGuest.UniqIds = uniqIds
		forceOfflineGuest.Data = data
		forceOfflineGuest.Delay = delay
		n.sendToSocketByUniqId(uniqIds[0], n.pack(netsvrProtocol.Cmd_ForceOfflineGuest, &forceOfflineGuest))
		return
	}
	group := n.getUniqIdsGroupByWorkerAddrAsHex(uniqIds)
	for workerAddrAsHex, currentUniqIds := range group {
		forceOfflineGuest := netsvrProtocol.ForceOfflineGuest{}
		forceOfflineGuest.UniqIds = currentUniqIds
		forceOfflineGuest.Data = data
		forceOfflineGuest.Delay = delay
		n.sendToSocketByWorkerAddrAsHex(workerAddrAsHex, n.pack(netsvrProtocol.Cmd_ForceOfflineGuest, &forceOfflineGuest))
	}
}

// CheckOnline 检查目标uniqId是否在线
func (n *NetBus) CheckOnline(uniqIds []string) *resp.CheckOnlineResp {
	ret := resp.CheckOnlineResp{Data: make(map[string]*netsvrProtocol.CheckOnlineResp)}
	if n.isSinglePoint() || len(uniqIds) == 1 {
		taskSocket := n.getTaskSocketByUniqId(uniqIds[0])
		if taskSocket == nil {
			return &ret
		}
		defer taskSocket.Release()
		checkOnlineReq := netsvrProtocol.CheckOnlineReq{}
		checkOnlineReq.UniqIds = uniqIds
		taskSocket.Send(n.pack(netsvrProtocol.Cmd_CheckOnline, &checkOnlineReq))
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::CheckOnline failed because the connection to the netsvr was disconnected")
			return &ret
		}
		checkOnlineResp := &netsvrProtocol.CheckOnlineResp{}
		if err := proto.Unmarshal(respData[4:], checkOnlineResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.CheckOnlineResp failed", "error", err)
			return &ret
		}
		ret.Data[taskSocket.GetWorkerAddr()] = checkOnlineResp
		return &ret
	}
	group := n.getUniqIdsGroupByWorkerAddrAsHex(uniqIds)
	fn := func(taskSocket *TaskSocket, currentUniqIds []string) {
		defer taskSocket.Release()
		checkOnlineReq := netsvrProtocol.CheckOnlineReq{}
		checkOnlineReq.UniqIds = currentUniqIds
		taskSocket.Send(n.pack(netsvrProtocol.Cmd_CheckOnline, &checkOnlineReq))
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::CheckOnline failed because the connection to the netsvr was disconnected")
			return
		}
		checkOnlineResp := &netsvrProtocol.CheckOnlineResp{}
		if err := proto.Unmarshal(respData[4:], checkOnlineResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.CheckOnlineResp failed", "error", err)
			return
		}
		ret.Data[taskSocket.GetWorkerAddr()] = checkOnlineResp
	}
	for workerAddrAsHex, currentUniqIds := range group {
		taskSocket := n.taskSocketPoolManger.GetSocket(workerAddrAsHex)
		if taskSocket == nil {
			continue
		}
		fn(taskSocket, currentUniqIds)
	}
	return &ret
}

// UniqIdList 获取所有网关中存储的uniqId
func (n *NetBus) UniqIdList() *resp.UniqIdListResp {
	ret := resp.UniqIdListResp{Data: make(map[string]*netsvrProtocol.UniqIdListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_UniqIdList, nil)
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::UniqIdList failed because the connection to the netsvr was disconnected")
			continue
		}
		uniqIdListResp := &netsvrProtocol.UniqIdListResp{}
		if err := proto.Unmarshal(respData[4:], uniqIdListResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.UniqIdListResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = uniqIdListResp
	}
	return &ret
}

// UniqIdCount 获取所有网关中存储的uniqId数量
func (n *NetBus) UniqIdCount() *resp.UniqIdCountResp {
	ret := resp.UniqIdCountResp{Data: make(map[string]*netsvrProtocol.UniqIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_UniqIdCount, nil)
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::UniqIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		uniqIdCountResp := &netsvrProtocol.UniqIdCountResp{}
		if err := proto.Unmarshal(respData[4:], uniqIdCountResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.UniqIdCountResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = uniqIdCountResp
	}
	return &ret
}

// TopicCount 获取所有网关中存储的topic数量
func (n *NetBus) TopicCount() *resp.TopicCountResp {
	ret := resp.TopicCountResp{Data: make(map[string]*netsvrProtocol.TopicCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicCount, nil)
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::TopicCount failed because the connection to the netsvr was disconnected")
			continue
		}
		topicCountResp := &netsvrProtocol.TopicCountResp{}
		if err := proto.Unmarshal(respData[4:], topicCountResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.TopicCountResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = topicCountResp
	}
	return &ret
}

// TopicList 获取所有网关中存储的topic
func (n *NetBus) TopicList() *resp.TopicListResp {
	ret := resp.TopicListResp{Data: make(map[string]*netsvrProtocol.TopicListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicList, nil)
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::TopicList failed because the connection to the netsvr was disconnected")
			continue
		}
		topicListResp := &netsvrProtocol.TopicListResp{}
		if err := proto.Unmarshal(respData[4:], topicListResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.TopicListResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = topicListResp
	}
	return &ret
}

// TopicUniqIdList 获取所有网关中存储的topic对应的uniqId
func (n *NetBus) TopicUniqIdList(topics []string) *resp.TopicUniqIdListResp {
	ret := resp.TopicUniqIdListResp{Data: make(map[string]*netsvrProtocol.TopicUniqIdListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicUniqIdList, &netsvrProtocol.TopicUniqIdListReq{Topics: topics})
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::TopicUniqIdList failed because the connection to the netsvr was disconnected")
			continue
		}
		topicUniqIdListResp := &netsvrProtocol.TopicUniqIdListResp{}
		if err := proto.Unmarshal(respData[4:], topicUniqIdListResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.TopicUniqIdListResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = topicUniqIdListResp
	}
	return &ret
}

// TopicUniqIdCount 获取所有网关中存储的topic对应的uniqId数量
func (n *NetBus) TopicUniqIdCount(topics []string, allTopic bool) *resp.TopicUniqIdCountResp {
	ret := resp.TopicUniqIdCountResp{Data: make(map[string]*netsvrProtocol.TopicUniqIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicUniqIdCount, &netsvrProtocol.TopicUniqIdCountReq{
		Topics:   topics,
		CountAll: allTopic,
	})
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::TopicUniqIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		topicUniqIdCountResp := &netsvrProtocol.TopicUniqIdCountResp{}
		if err := proto.Unmarshal(respData[4:], topicUniqIdCountResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.TopicUniqIdCountResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = topicUniqIdCountResp
	}
	return &ret
}

// TopicCustomerIdList 获取所有网关中存储的topic对应的customerId
func (n *NetBus) TopicCustomerIdList(topics []string) *resp.TopicCustomerIdListResp {
	ret := resp.TopicCustomerIdListResp{Data: make(map[string]*netsvrProtocol.TopicCustomerIdListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicCustomerIdList, &netsvrProtocol.TopicCustomerIdListReq{Topics: topics})
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::TopicCustomerIdList failed because the connection to the netsvr was disconnected")
			continue
		}
		topicCustomerIdListResp := &netsvrProtocol.TopicCustomerIdListResp{}
		if err := proto.Unmarshal(respData[4:], topicCustomerIdListResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.TopicCustomerIdListResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = topicCustomerIdListResp
	}
	return &ret
}

// TopicCustomerIdToUniqIdsList 获取所有网关中存储的topic对应的customerId对应的uniqId
func (n *NetBus) TopicCustomerIdToUniqIdsList(topics []string) *resp.TopicCustomerIdToUniqIdsListResp {
	ret := resp.TopicCustomerIdToUniqIdsListResp{Data: make(map[string]*netsvrProtocol.TopicCustomerIdToUniqIdsListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicCustomerIdToUniqIdsList, &netsvrProtocol.TopicCustomerIdToUniqIdsListReq{Topics: topics})
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::TopicCustomerIdToUniqIdsList failed because the connection to the netsvr was disconnected")
			continue
		}
		topicCustomerIdToUniqIdsListResp := &netsvrProtocol.TopicCustomerIdToUniqIdsListResp{}
		if err := proto.Unmarshal(respData[4:], topicCustomerIdToUniqIdsListResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.TopicCustomerIdToUniqIdsListResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = topicCustomerIdToUniqIdsListResp
	}
	return &ret
}

// TopicCustomerIdCount 获取所有网关中存储的topic对应的customerId数量
func (n *NetBus) TopicCustomerIdCount(topics []string, allTopic bool) *resp.TopicCustomerIdCountResp {
	ret := resp.TopicCustomerIdCountResp{Data: make(map[string]*netsvrProtocol.TopicCustomerIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicCustomerIdCount, &netsvrProtocol.TopicCustomerIdCountReq{
		Topics:   topics,
		CountAll: allTopic,
	})
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::TopicCustomerIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		topicCustomerIdCountResp := &netsvrProtocol.TopicCustomerIdCountResp{}
		if err := proto.Unmarshal(respData[4:], topicCustomerIdCountResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.TopicCustomerIdCountResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = topicCustomerIdCountResp
	}
	return &ret
}

// ConnInfo 获取所有网关中存储的连接信息
func (n *NetBus) ConnInfo(uniqIds []string, reqCustomerId bool, reqSession bool, reqTopic bool) *resp.ConnInfoResp {
	ret := resp.ConnInfoResp{Data: make(map[string]*netsvrProtocol.ConnInfoResp)}
	if n.isSinglePoint() || len(uniqIds) == 1 {
		taskSocket := n.getTaskSocketByUniqId(uniqIds[0])
		if taskSocket == nil {
			return &ret
		}
		defer taskSocket.Release()
		connInfoReq := netsvrProtocol.ConnInfoReq{
			UniqIds:       uniqIds,
			ReqCustomerId: reqCustomerId,
			ReqSession:    reqSession,
			ReqTopic:      reqTopic,
		}
		taskSocket.Send(n.pack(netsvrProtocol.Cmd_ConnInfo, &connInfoReq))
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::ConnInfo failed because the connection to the netsvr was disconnected")
			return &ret
		}
		connInfoResp := &netsvrProtocol.ConnInfoResp{}
		if err := proto.Unmarshal(respData[4:], connInfoResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.ConnInfoResp failed", "error", err)
			return &ret
		}
		ret.Data[taskSocket.GetWorkerAddr()] = connInfoResp
		return &ret
	}
	group := n.getUniqIdsGroupByWorkerAddrAsHex(uniqIds)
	fn := func(taskSocket *TaskSocket, currentUniqIds []string) {
		defer taskSocket.Release()
		connInfoReq := netsvrProtocol.ConnInfoReq{
			UniqIds:       currentUniqIds,
			ReqCustomerId: reqCustomerId,
			ReqSession:    reqSession,
			ReqTopic:      reqTopic,
		}
		taskSocket.Send(n.pack(netsvrProtocol.Cmd_ConnInfo, &connInfoReq))
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::ConnInfo failed because the connection to the netsvr was disconnected")
			return
		}
		connInfoResp := &netsvrProtocol.ConnInfoResp{}
		if err := proto.Unmarshal(respData[4:], connInfoResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.ConnInfoResp failed", "error", err)
			return
		}
		ret.Data[taskSocket.GetWorkerAddr()] = connInfoResp
	}
	for workerAddrAsHex, currentUniqIds := range group {
		taskSocket := n.taskSocketPoolManger.GetSocket(workerAddrAsHex)
		if taskSocket == nil {
			continue
		}
		fn(taskSocket, currentUniqIds)
	}
	return &ret
}

// ConnInfoByCustomerId 根据customerId获取所有网关中存储的连接信息
func (n *NetBus) ConnInfoByCustomerId(customerIds []string, reqUniqId bool, reqSession bool, reqTopic bool) *resp.ConnInfoByCustomerIdResp {
	ret := resp.ConnInfoByCustomerIdResp{Data: make(map[string]*netsvrProtocol.ConnInfoByCustomerIdResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_ConnInfoByCustomerId, &netsvrProtocol.ConnInfoByCustomerIdReq{
		CustomerIds: customerIds,
		ReqUniqId:   reqUniqId,
		ReqSession:  reqSession,
		ReqTopic:    reqTopic,
	})
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::ConnInfoByCustomerId failed because the connection to the netsvr was disconnected")
			continue
		}
		connInfoByCustomerIdResp := &netsvrProtocol.ConnInfoByCustomerIdResp{}
		if err := proto.Unmarshal(respData[4:], connInfoByCustomerIdResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.ConnInfoByCustomerIdResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = connInfoByCustomerIdResp
	}
	return &ret
}

// Metrics 获取所有网关的统计信息
func (n *NetBus) Metrics() *resp.MetricsResp {
	ret := resp.MetricsResp{Data: make(map[string]*netsvrProtocol.MetricsResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_Metrics, nil)
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::Metrics failed because the connection to the netsvr was disconnected")
			continue
		}
		metricsResp := &netsvrProtocol.MetricsResp{}
		if err := proto.Unmarshal(respData[4:], metricsResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.MetricsResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = metricsResp
	}
	return &ret
}

// Limit 设置或读取网关针对business的每秒转发数量的限制的配置
func (n *NetBus) Limit(limitReq *netsvrProtocol.LimitReq, workerAddr string) *resp.LimitResp {
	ret := resp.LimitResp{Data: make(map[string]*netsvrProtocol.LimitResp)}
	var taskSockets []*TaskSocket
	if workerAddr == "" {
		taskSockets = n.taskSocketPoolManger.GetSockets()
	} else {
		workerAddrAsHex := WorkerAddrConvertToHex(workerAddr)
		taskSocket := n.taskSocketPoolManger.GetSocket(workerAddrAsHex)
		if taskSocket == nil {
			return nil
		}
		taskSockets = []*TaskSocket{taskSocket}
	}
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_Limit, limitReq)
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::Limit failed because the connection to the netsvr was disconnected")
			continue
		}
		limitResp := &netsvrProtocol.LimitResp{}
		if err := proto.Unmarshal(respData[4:], limitResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.LimitResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = limitResp
	}
	return &ret
}

// CustomerIdList 获取所有网关的customerId列表
func (n *NetBus) CustomerIdList() *resp.CustomerIdListResp {
	ret := resp.CustomerIdListResp{Data: make(map[string]*netsvrProtocol.CustomerIdListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_CustomerIdList, nil)
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::CustomerIdList failed because the connection to the netsvr was disconnected")
			continue
		}
		customerIdListResp := &netsvrProtocol.CustomerIdListResp{}
		if err := proto.Unmarshal(respData[4:], customerIdListResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.CustomerIdListResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = customerIdListResp
	}
	return &ret
}

// CustomerIdCount 统计网关的在线客户数，注意各个网关的客户数之和不一定等于总在线客户数，因为可能一个客户有多个设备连接到不同网关
func (n *NetBus) CustomerIdCount() *resp.CustomerIdCountResp {
	ret := resp.CustomerIdCountResp{Data: make(map[string]*netsvrProtocol.CustomerIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_CustomerIdCount, nil)
	for _, taskSocket := range taskSockets {
		taskSocket.Send(message)
		respData := taskSocket.Receive()
		if respData == nil {
			logger.Error("call Cmd::CustomerIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		customerIdCountResp := &netsvrProtocol.CustomerIdCountResp{}
		if err := proto.Unmarshal(respData[4:], customerIdCountResp); err != nil {
			logger.Error("unmarshal netsvrProtocol.CustomerIdCountResp failed", "error", err)
			continue
		}
		ret.Data[taskSocket.GetWorkerAddr()] = customerIdCountResp
	}
	return &ret
}

func (n *NetBus) sendToSockets(data []byte) {
	//优先用mainSocket发送，因为mainSocket是异步发送的，而且网关可以为mainSocket开启多条协程处理这些信息
	if n.mainSocketManager != nil {
		mainSockets := n.mainSocketManager.GetSockets()
		if len(mainSockets) > 0 {
			for _, mainSocket := range mainSockets {
				mainSocket.Send(data)
			}
			return
		}
	}
	//如果mainSocket不存在，再通过taskSocket发送
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, taskSocket := range taskSockets {
			taskSocket.Release()
		}
	}()
	for _, taskSocket := range taskSockets {
		taskSocket.Send(data)
	}
}

func (n *NetBus) sendToSocketByUniqId(uniqId string, data []byte) {
	//优先用mainSocket发送，因为mainSocket是异步发送的，而且网关可以为mainSocket开启多条协程处理这些信息
	if n.mainSocketManager != nil {
		mainSocket := n.mainSocketManager.GetSocket(UniqIdConvertToWorkerAddrAsHex(uniqId))
		if mainSocket != nil {
			mainSocket.Send(data)
			return
		}
	}
	//如果mainSocket不存在，再通过taskSocket发送
	taskSocket := n.taskSocketPoolManger.GetSocket(UniqIdConvertToWorkerAddrAsHex(uniqId))
	if taskSocket != nil {
		defer taskSocket.Release()
		taskSocket.Send(data)
		return
	}
}

func (n *NetBus) sendToSocketByWorkerAddrAsHex(workerAddrAsHex string, data []byte) {
	//优先用mainSocket发送，因为mainSocket是异步发送的，而且网关可以为mainSocket开启多条协程处理这些信息
	if n.mainSocketManager != nil {
		mainSocket := n.mainSocketManager.GetSocket(workerAddrAsHex)
		if mainSocket != nil {
			mainSocket.Send(data)
			return
		}
	}
	//如果mainSocket不存在，再通过taskSocket发送
	taskSocket := n.taskSocketPoolManger.GetSocket(workerAddrAsHex)
	if taskSocket != nil {
		defer taskSocket.Release()
		taskSocket.Send(data)
	}
}

func (n *NetBus) getTaskSocketByUniqId(uniqId string) *TaskSocket {
	return n.taskSocketPoolManger.GetSocket(UniqIdConvertToWorkerAddrAsHex(uniqId))
}

func (n *NetBus) isSinglePoint() bool {
	return n.taskSocketPoolManger.Count() == 1
}

// getUniqIdsGroupByWorkerAddrAsHex 根据uniqId列表分组，返回每个workerAddrAsHex对应的uniqId列表
func (n *NetBus) getUniqIdsGroupByWorkerAddrAsHex(uniqIds []string) map[string][]string {
	ret := make(map[string][]string)
	for _, uniqId := range uniqIds {
		workerAddrAsHex := UniqIdConvertToWorkerAddrAsHex(uniqId)
		ret[workerAddrAsHex] = append(ret[workerAddrAsHex], uniqId)
	}
	return ret
}

func (n *NetBus) pack(cmd netsvrProtocol.Cmd, req proto.Message) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data[0:4], uint32(cmd))
	if req == nil {
		return data
	}
	var err error
	data, err = (proto.MarshalOptions{}).MarshalAppend(data, req)
	if err != nil {
		logger.Error(fmt.Sprintf("Proto marshal %T failed", req), "error", err)
		return nil
	}
	return data
}
