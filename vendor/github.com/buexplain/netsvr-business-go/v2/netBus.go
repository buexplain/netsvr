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
	"github.com/buexplain/netsvr-business-go/v2/contract"
	"github.com/buexplain/netsvr-business-go/v2/log"
	"github.com/buexplain/netsvr-business-go/v2/ret"
	"github.com/buexplain/netsvr-business-go/v2/taskSocket"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"google.golang.org/protobuf/proto"
)

type NetBus struct {
	taskSocketPoolManger *taskSocket.Manger
}

func NewNetBus(taskSocketPoolManger *taskSocket.Manger) *NetBus {
	if taskSocketPoolManger == nil {
		panic("taskSocketPoolManger is nil")
	}
	return &NetBus{
		taskSocketPoolManger: taskSocketPoolManger,
	}
}

// Close 关闭网关
func (n *NetBus) Close() {
	n.taskSocketPoolManger.Close()
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
	group := n.getUniqIdsGroupByAddrAsHex(uniqIds)
	for addrAsHex, currentUniqIds := range group {
		multicast := netsvrProtocol.Multicast{
			UniqIds: currentUniqIds,
			Data:    data,
		}
		message := n.pack(netsvrProtocol.Cmd_Multicast, &multicast)
		n.sendToSocketByAddrAsHex(addrAsHex, message)
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
		addrAsHex := contract.UniqIdConvertToAddrAsHex(uniqId)
		if b, ok := bulks[addrAsHex]; ok {
			b.uniqIds = append(b.uniqIds, uniqId)
			b.data = append(b.data, data[index])
		} else {
			b := &bulk{
				uniqIds: []string{uniqId},
				data:    [][]byte{data[index]},
			}
			bulks[addrAsHex] = b
		}
	}
	//分组完毕，循环发送到各个网关
	for addrAsHex, b := range bulks {
		singleCastBulk := netsvrProtocol.SingleCastBulk{}
		singleCastBulk.Data = b.data
		singleCastBulk.UniqIds = b.uniqIds
		n.sendToSocketByAddrAsHex(addrAsHex, n.pack(netsvrProtocol.Cmd_SingleCastBulk, &singleCastBulk))
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
	group := n.getUniqIdsGroupByAddrAsHex(uniqIds)
	for addrAsHex, currentUniqIds := range group {
		forceOffline := netsvrProtocol.ForceOffline{}
		forceOffline.UniqIds = currentUniqIds
		forceOffline.Data = data
		n.sendToSocketByAddrAsHex(addrAsHex, n.pack(netsvrProtocol.Cmd_ForceOffline, &forceOffline))
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
	group := n.getUniqIdsGroupByAddrAsHex(uniqIds)
	for addrAsHex, currentUniqIds := range group {
		forceOfflineGuest := netsvrProtocol.ForceOfflineGuest{}
		forceOfflineGuest.UniqIds = currentUniqIds
		forceOfflineGuest.Data = data
		forceOfflineGuest.Delay = delay
		n.sendToSocketByAddrAsHex(addrAsHex, n.pack(netsvrProtocol.Cmd_ForceOfflineGuest, &forceOfflineGuest))
	}
}

// CheckOnline 检查目标uniqId是否在线
func (n *NetBus) CheckOnline(uniqIds []string) *ret.CheckOnlineRet {
	res := ret.CheckOnlineRet{Data: make(map[string]*netsvrProtocol.CheckOnlineResp)}
	if n.isSinglePoint() || len(uniqIds) == 1 {
		socket := n.getTaskSocketByUniqId(uniqIds[0])
		if socket == nil {
			return &res
		}
		defer socket.Release()
		checkOnlineReq := netsvrProtocol.CheckOnlineReq{}
		checkOnlineReq.UniqIds = uniqIds
		socket.Send(n.pack(netsvrProtocol.Cmd_CheckOnline, &checkOnlineReq))
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::CheckOnline failed because the connection to the netsvr was disconnected")
			return &res
		}
		checkOnlineResp := &netsvrProtocol.CheckOnlineResp{}
		if err := proto.Unmarshal(respData[4:], checkOnlineResp); err != nil {
			log.Error("unmarshal netsvrProtocol.CheckOnlineResp failed", "error", err)
			return &res
		}
		res.Data[socket.GetAddr()] = checkOnlineResp
		return &res
	}
	group := n.getUniqIdsGroupByAddrAsHex(uniqIds)
	fn := func(socket *taskSocket.TaskSocket, currentUniqIds []string) {
		defer socket.Release()
		checkOnlineReq := netsvrProtocol.CheckOnlineReq{}
		checkOnlineReq.UniqIds = currentUniqIds
		socket.Send(n.pack(netsvrProtocol.Cmd_CheckOnline, &checkOnlineReq))
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::CheckOnline failed because the connection to the netsvr was disconnected")
			return
		}
		checkOnlineResp := &netsvrProtocol.CheckOnlineResp{}
		if err := proto.Unmarshal(respData[4:], checkOnlineResp); err != nil {
			log.Error("unmarshal netsvrProtocol.CheckOnlineResp failed", "error", err)
			return
		}
		res.Data[socket.GetAddr()] = checkOnlineResp
	}
	for addrAsHex, currentUniqIds := range group {
		socket := n.taskSocketPoolManger.GetSocket(addrAsHex)
		if socket == nil {
			continue
		}
		fn(socket, currentUniqIds)
	}
	return &res
}

// UniqIdList 获取所有网关中存储的uniqId
func (n *NetBus) UniqIdList() *ret.UniqIdListRet {
	res := ret.UniqIdListRet{Data: make(map[string]*netsvrProtocol.UniqIdListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_UniqIdList, nil)
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::UniqIdList failed because the connection to the netsvr was disconnected")
			continue
		}
		uniqIdListResp := &netsvrProtocol.UniqIdListResp{}
		if err := proto.Unmarshal(respData[4:], uniqIdListResp); err != nil {
			log.Error("unmarshal netsvrProtocol.UniqIdListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = uniqIdListResp
	}
	return &res
}

// UniqIdCount 获取所有网关中存储的uniqId数量
func (n *NetBus) UniqIdCount() *ret.UniqIdCountRet {
	res := ret.UniqIdCountRet{Data: make(map[string]*netsvrProtocol.UniqIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_UniqIdCount, nil)
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::UniqIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		uniqIdCountResp := &netsvrProtocol.UniqIdCountResp{}
		if err := proto.Unmarshal(respData[4:], uniqIdCountResp); err != nil {
			log.Error("unmarshal netsvrProtocol.UniqIdCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = uniqIdCountResp
	}
	return &res
}

// TopicCount 获取所有网关中存储的topic数量
func (n *NetBus) TopicCount() *ret.TopicCountRet {
	res := ret.TopicCountRet{Data: make(map[string]*netsvrProtocol.TopicCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicCount, nil)
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::TopicCount failed because the connection to the netsvr was disconnected")
			continue
		}
		topicCountResp := &netsvrProtocol.TopicCountResp{}
		if err := proto.Unmarshal(respData[4:], topicCountResp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = topicCountResp
	}
	return &res
}

// TopicList 获取所有网关中存储的topic
func (n *NetBus) TopicList() *ret.TopicListRet {
	res := ret.TopicListRet{Data: make(map[string]*netsvrProtocol.TopicListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicList, nil)
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::TopicList failed because the connection to the netsvr was disconnected")
			continue
		}
		topicListResp := &netsvrProtocol.TopicListResp{}
		if err := proto.Unmarshal(respData[4:], topicListResp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = topicListResp
	}
	return &res
}

// TopicUniqIdList 获取所有网关中存储的topic对应的uniqId
func (n *NetBus) TopicUniqIdList(topics []string) *ret.TopicUniqIdListRet {
	res := ret.TopicUniqIdListRet{Data: make(map[string]*netsvrProtocol.TopicUniqIdListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicUniqIdList, &netsvrProtocol.TopicUniqIdListReq{Topics: topics})
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::TopicUniqIdList failed because the connection to the netsvr was disconnected")
			continue
		}
		topicUniqIdListResp := &netsvrProtocol.TopicUniqIdListResp{}
		if err := proto.Unmarshal(respData[4:], topicUniqIdListResp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicUniqIdListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = topicUniqIdListResp
	}
	return &res
}

// TopicUniqIdCount 获取所有网关中存储的topic对应的uniqId数量
func (n *NetBus) TopicUniqIdCount(topics []string, allTopic bool) *ret.TopicUniqIdCountRet {
	res := ret.TopicUniqIdCountRet{Data: make(map[string]*netsvrProtocol.TopicUniqIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicUniqIdCount, &netsvrProtocol.TopicUniqIdCountReq{
		Topics:   topics,
		CountAll: allTopic,
	})
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::TopicUniqIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		topicUniqIdCountResp := &netsvrProtocol.TopicUniqIdCountResp{}
		if err := proto.Unmarshal(respData[4:], topicUniqIdCountResp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicUniqIdCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = topicUniqIdCountResp
	}
	return &res
}

// TopicCustomerIdList 获取所有网关中存储的topic对应的customerId
func (n *NetBus) TopicCustomerIdList(topics []string) *ret.TopicCustomerIdListRet {
	res := ret.TopicCustomerIdListRet{Data: make(map[string]*netsvrProtocol.TopicCustomerIdListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicCustomerIdList, &netsvrProtocol.TopicCustomerIdListReq{Topics: topics})
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::TopicCustomerIdList failed because the connection to the netsvr was disconnected")
			continue
		}
		topicCustomerIdListResp := &netsvrProtocol.TopicCustomerIdListResp{}
		if err := proto.Unmarshal(respData[4:], topicCustomerIdListResp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicCustomerIdListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = topicCustomerIdListResp
	}
	return &res
}

// TopicCustomerIdToUniqIdsList 获取所有网关中存储的topic对应的customerId对应的uniqId
func (n *NetBus) TopicCustomerIdToUniqIdsList(topics []string) *ret.TopicCustomerIdToUniqIdsListRet {
	res := ret.TopicCustomerIdToUniqIdsListRet{Data: make(map[string]*netsvrProtocol.TopicCustomerIdToUniqIdsListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicCustomerIdToUniqIdsList, &netsvrProtocol.TopicCustomerIdToUniqIdsListReq{Topics: topics})
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::TopicCustomerIdToUniqIdsList failed because the connection to the netsvr was disconnected")
			continue
		}
		topicCustomerIdToUniqIdsListResp := &netsvrProtocol.TopicCustomerIdToUniqIdsListResp{}
		if err := proto.Unmarshal(respData[4:], topicCustomerIdToUniqIdsListResp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicCustomerIdToUniqIdsListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = topicCustomerIdToUniqIdsListResp
	}
	return &res
}

// TopicCustomerIdCount 获取所有网关中存储的topic对应的customerId数量
func (n *NetBus) TopicCustomerIdCount(topics []string, allTopic bool) *ret.TopicCustomerIdCountRet {
	res := ret.TopicCustomerIdCountRet{Data: make(map[string]*netsvrProtocol.TopicCustomerIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicCustomerIdCount, &netsvrProtocol.TopicCustomerIdCountReq{
		Topics:   topics,
		CountAll: allTopic,
	})
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::TopicCustomerIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		topicCustomerIdCountResp := &netsvrProtocol.TopicCustomerIdCountResp{}
		if err := proto.Unmarshal(respData[4:], topicCustomerIdCountResp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicCustomerIdCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = topicCustomerIdCountResp
	}
	return &res
}

// ConnInfo 获取所有网关中存储的连接信息
func (n *NetBus) ConnInfo(uniqIds []string, reqCustomerId bool, reqSession bool, reqTopic bool) *ret.ConnInfoRet {
	res := ret.ConnInfoRet{Data: make(map[string]*netsvrProtocol.ConnInfoResp)}
	if n.isSinglePoint() || len(uniqIds) == 1 {
		socket := n.getTaskSocketByUniqId(uniqIds[0])
		if socket == nil {
			return &res
		}
		defer socket.Release()
		connInfoReq := netsvrProtocol.ConnInfoReq{
			UniqIds:       uniqIds,
			ReqCustomerId: reqCustomerId,
			ReqSession:    reqSession,
			ReqTopic:      reqTopic,
		}
		socket.Send(n.pack(netsvrProtocol.Cmd_ConnInfo, &connInfoReq))
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::ConnInfo failed because the connection to the netsvr was disconnected")
			return &res
		}
		connInfoResp := &netsvrProtocol.ConnInfoResp{}
		if err := proto.Unmarshal(respData[4:], connInfoResp); err != nil {
			log.Error("unmarshal netsvrProtocol.ConnInfoResp failed", "error", err)
			return &res
		}
		res.Data[socket.GetAddr()] = connInfoResp
		return &res
	}
	group := n.getUniqIdsGroupByAddrAsHex(uniqIds)
	fn := func(socket *taskSocket.TaskSocket, currentUniqIds []string) {
		defer socket.Release()
		connInfoReq := netsvrProtocol.ConnInfoReq{
			UniqIds:       currentUniqIds,
			ReqCustomerId: reqCustomerId,
			ReqSession:    reqSession,
			ReqTopic:      reqTopic,
		}
		socket.Send(n.pack(netsvrProtocol.Cmd_ConnInfo, &connInfoReq))
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::ConnInfo failed because the connection to the netsvr was disconnected")
			return
		}
		connInfoResp := &netsvrProtocol.ConnInfoResp{}
		if err := proto.Unmarshal(respData[4:], connInfoResp); err != nil {
			log.Error("unmarshal netsvrProtocol.ConnInfoResp failed", "error", err)
			return
		}
		res.Data[socket.GetAddr()] = connInfoResp
	}
	for addrAsHex, currentUniqIds := range group {
		socket := n.taskSocketPoolManger.GetSocket(addrAsHex)
		if socket == nil {
			continue
		}
		fn(socket, currentUniqIds)
	}
	return &res
}

// ConnInfoByCustomerId 根据customerId获取所有网关中存储的连接信息
func (n *NetBus) ConnInfoByCustomerId(customerIds []string, reqUniqId bool, reqSession bool, reqTopic bool) *ret.ConnInfoByCustomerIdRet {
	res := ret.ConnInfoByCustomerIdRet{Data: make(map[string]*netsvrProtocol.ConnInfoByCustomerIdResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_ConnInfoByCustomerId, &netsvrProtocol.ConnInfoByCustomerIdReq{
		CustomerIds: customerIds,
		ReqUniqId:   reqUniqId,
		ReqSession:  reqSession,
		ReqTopic:    reqTopic,
	})
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::ConnInfoByCustomerId failed because the connection to the netsvr was disconnected")
			continue
		}
		connInfoByCustomerIdResp := &netsvrProtocol.ConnInfoByCustomerIdResp{}
		if err := proto.Unmarshal(respData[4:], connInfoByCustomerIdResp); err != nil {
			log.Error("unmarshal netsvrProtocol.ConnInfoByCustomerIdResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = connInfoByCustomerIdResp
	}
	return &res
}

// Metrics 获取所有网关的统计信息
func (n *NetBus) Metrics() *ret.MetricsRet {
	res := ret.MetricsRet{Data: make(map[string]*netsvrProtocol.MetricsResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_Metrics, nil)
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::Metrics failed because the connection to the netsvr was disconnected")
			continue
		}
		metricsResp := &netsvrProtocol.MetricsResp{}
		if err := proto.Unmarshal(respData[4:], metricsResp); err != nil {
			log.Error("unmarshal netsvrProtocol.MetricsResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = metricsResp
	}
	return &res
}

// Limit 设置或读取网关针对business的每秒转发数量的限制的配置
func (n *NetBus) Limit(limitReq *netsvrProtocol.LimitReq, addr string) *ret.LimitRet {
	res := ret.LimitRet{Data: make(map[string]*netsvrProtocol.LimitResp)}
	var taskSockets []*taskSocket.TaskSocket
	if addr == "" {
		taskSockets = n.taskSocketPoolManger.GetSockets()
	} else {
		addrAsHex := contract.AddrConvertToHex(addr)
		socket := n.taskSocketPoolManger.GetSocket(addrAsHex)
		if socket == nil {
			return nil
		}
		taskSockets = []*taskSocket.TaskSocket{socket}
	}
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_Limit, limitReq)
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::Limit failed because the connection to the netsvr was disconnected")
			continue
		}
		limitResp := &netsvrProtocol.LimitResp{}
		if err := proto.Unmarshal(respData[4:], limitResp); err != nil {
			log.Error("unmarshal netsvrProtocol.LimitResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = limitResp
	}
	return &res
}

// CustomerIdList 获取所有网关的customerId列表
func (n *NetBus) CustomerIdList() *ret.CustomerIdListRet {
	res := ret.CustomerIdListRet{Data: make(map[string]*netsvrProtocol.CustomerIdListResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_CustomerIdList, nil)
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::CustomerIdList failed because the connection to the netsvr was disconnected")
			continue
		}
		customerIdListResp := &netsvrProtocol.CustomerIdListResp{}
		if err := proto.Unmarshal(respData[4:], customerIdListResp); err != nil {
			log.Error("unmarshal netsvrProtocol.CustomerIdListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = customerIdListResp
	}
	return &res
}

// CustomerIdCount 统计网关的在线客户数，注意各个网关的客户数之和不一定等于总在线客户数，因为可能一个客户有多个设备连接到不同网关
func (n *NetBus) CustomerIdCount() *ret.CustomerIdCountRet {
	res := ret.CustomerIdCountRet{Data: make(map[string]*netsvrProtocol.CustomerIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_CustomerIdCount, nil)
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::CustomerIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		customerIdCountResp := &netsvrProtocol.CustomerIdCountResp{}
		if err := proto.Unmarshal(respData[4:], customerIdCountResp); err != nil {
			log.Error("unmarshal netsvrProtocol.CustomerIdCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = customerIdCountResp
	}
	return &res
}

func (n *NetBus) sendToSockets(data []byte) {
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	for _, socket := range taskSockets {
		socket.Send(data)
	}
}

func (n *NetBus) sendToSocketByUniqId(uniqId string, data []byte) {
	socket := n.taskSocketPoolManger.GetSocket(contract.UniqIdConvertToAddrAsHex(uniqId))
	if socket != nil {
		defer socket.Release()
		socket.Send(data)
		return
	}
}

func (n *NetBus) sendToSocketByAddrAsHex(addrAsHex string, data []byte) {
	socket := n.taskSocketPoolManger.GetSocket(addrAsHex)
	if socket != nil {
		defer socket.Release()
		socket.Send(data)
	}
}

func (n *NetBus) getTaskSocketByUniqId(uniqId string) *taskSocket.TaskSocket {
	return n.taskSocketPoolManger.GetSocket(contract.UniqIdConvertToAddrAsHex(uniqId))
}

func (n *NetBus) isSinglePoint() bool {
	return n.taskSocketPoolManger.Count() == 1
}

// getUniqIdsGroupByAddrAsHex 根据uniqId列表分组，返回每个addrAsHex对应的uniqId列表
func (n *NetBus) getUniqIdsGroupByAddrAsHex(uniqIds []string) map[string][]string {
	res := make(map[string][]string)
	for _, uniqId := range uniqIds {
		addrAsHex := contract.UniqIdConvertToAddrAsHex(uniqId)
		res[addrAsHex] = append(res[addrAsHex], uniqId)
	}
	return res
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
		log.Error(fmt.Sprintf("Proto marshal %T failed", req), "error", err)
		return nil
	}
	return data
}
