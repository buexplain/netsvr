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

// Package netsvrBusiness 是业务进程与网关交互的 SDK。
// 方法集与协议一一对应，并额外提供若干语义化便捷方法；入参与返回值直接使用协议生成类型。
package netsvrBusiness

import (
	"encoding/binary"
	"fmt"

	"github.com/buexplain/netsvr-business-go/v3/contract"
	"github.com/buexplain/netsvr-business-go/v3/log"
	"github.com/buexplain/netsvr-business-go/v3/ret"
	"github.com/buexplain/netsvr-business-go/v3/taskSocket"
	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
	"google.golang.org/protobuf/proto"
)

// NetBus 业务进程与网关交互的入口，封装了全部「发送」与「查询」指令。
// 所有方法并发安全，可多协程共用同一个实例
type NetBus struct {
	taskSocketPoolManger *taskSocket.Manger
}

// NewNetBus 创建 NetBus，必传 task socket 连接池管理器
func NewNetBus(taskSocketPoolManger *taskSocket.Manger) *NetBus {
	if taskSocketPoolManger == nil {
		panic("taskSocketPoolManger is nil")
	}
	return &NetBus{
		taskSocketPoolManger: taskSocketPoolManger,
	}
}

// Close 关闭 SDK
func (n *NetBus) Close() {
	n.taskSocketPoolManger.Close()
}

// ============================== 发送：全量广播 ==============================

// BroadcastBulk 批量广播，网关按顺序把每一条数据广播给全部连接
func (n *NetBus) BroadcastBulk(data [][]byte) {
	if len(data) == 0 {
		return
	}
	n.sendToSockets(n.pack(netsvrProtocol.Cmd_BroadcastBulk, &netsvrProtocol.BroadcastBulk{Data: data}))
}

// Broadcast 广播一条数据给全部连接，等价于 BroadcastBulk 传一条数据
func (n *NetBus) Broadcast(data []byte) {
	n.BroadcastBulk([][]byte{data})
}

// ============================== 发送：按 uniqId ==============================

// SingleCastBulk 按uniqId批量单播。每一项是一组uniqId与其数据，
// 网关会把本项内每一条数据按顺序发给本项内的每一个uniqId。
func (n *NetBus) SingleCastBulk(items []*netsvrProtocol.SingleCastBulkItem) {
	if len(items) == 0 {
		return
	}
	//网关是单机部署，则直接发送
	if n.isSinglePoint() {
		n.sendToSockets(n.pack(netsvrProtocol.Cmd_SingleCastBulk, &netsvrProtocol.SingleCastBulk{Items: items}))
		return
	}
	//网关是多机器部署，按每个uniqId所在网关拆分，再分别发送到对应网关
	bulks := make(map[string][]*netsvrProtocol.SingleCastBulkItem)
	for _, item := range items {
		for _, uniqId := range item.GetUniqIds() {
			addrAsHex := contract.UniqIdConvertToAddrAsHex(uniqId)
			bulks[addrAsHex] = append(bulks[addrAsHex], &netsvrProtocol.SingleCastBulkItem{
				UniqIds: []string{uniqId},
				Data:    item.GetData(),
			})
		}
	}
	for addrAsHex, currentItems := range bulks {
		n.sendToSocketByAddrAsHex(addrAsHex, n.pack(netsvrProtocol.Cmd_SingleCastBulk, &netsvrProtocol.SingleCastBulk{Items: currentItems}))
	}
}

// SendToUniqId 给一个连接发送一条数据
func (n *NetBus) SendToUniqId(uniqId string, data []byte) {
	n.SingleCastBulk([]*netsvrProtocol.SingleCastBulkItem{
		{UniqIds: []string{uniqId}, Data: [][]byte{data}},
	})
}

// SendToUniqIds 给一组连接发送同一条数据
func (n *NetBus) SendToUniqIds(uniqIds []string, data []byte) {
	n.SingleCastBulk([]*netsvrProtocol.SingleCastBulkItem{
		{UniqIds: uniqIds, Data: [][]byte{data}},
	})
}

// ============================== 发送：按 customerId ==============================

// SingleCastBulkByCustomerId 按customerId批量单播。每一项是一组customerId与其数据，
// 网关会把本项内每一条数据按顺序发给本项内每一个customerId对应的所有连接。
func (n *NetBus) SingleCastBulkByCustomerId(items []*netsvrProtocol.SingleCastBulkByCustomerIdItem) {
	if len(items) == 0 {
		return
	}
	n.sendToSockets(n.pack(netsvrProtocol.Cmd_SingleCastBulkByCustomerId, &netsvrProtocol.SingleCastBulkByCustomerId{Items: items}))
}

// SendToCustomerId 给一个客户的所有连接发送一条数据
func (n *NetBus) SendToCustomerId(customerId string, data []byte) {
	n.SingleCastBulkByCustomerId([]*netsvrProtocol.SingleCastBulkByCustomerIdItem{
		{CustomerIds: []string{customerId}, Data: [][]byte{data}},
	})
}

// SendToCustomerIds 给一组客户的所有连接发送同一条数据
func (n *NetBus) SendToCustomerIds(customerIds []string, data []byte) {
	n.SingleCastBulkByCustomerId([]*netsvrProtocol.SingleCastBulkByCustomerIdItem{
		{CustomerIds: customerIds, Data: [][]byte{data}},
	})
}

// ============================== 发送：按 topic ==============================

// TopicPublishBulk 批量发布。每一项是一组主题与其数据，
// 网关会把本项内每一条数据按顺序发布给本项内每一个主题的所有订阅连接。
func (n *NetBus) TopicPublishBulk(items []*netsvrProtocol.TopicPublishBulkItem) {
	if len(items) == 0 {
		return
	}
	n.sendToSockets(n.pack(netsvrProtocol.Cmd_TopicPublishBulk, &netsvrProtocol.TopicPublishBulk{Items: items}))
}

// PublishToTopic 给一个主题发布一条数据
func (n *NetBus) PublishToTopic(topic string, data []byte) {
	n.TopicPublishBulk([]*netsvrProtocol.TopicPublishBulkItem{
		{Topics: []string{topic}, Data: [][]byte{data}},
	})
}

// PublishToTopics 给一组主题发布同一条数据
func (n *NetBus) PublishToTopics(topics []string, data []byte) {
	n.TopicPublishBulk([]*netsvrProtocol.TopicPublishBulkItem{
		{Topics: topics, Data: [][]byte{data}},
	})
}

// ============================== 发送：连接信息与订阅 ==============================

// ConnInfoUpdate 更新连接存储在网关中的信息
func (n *NetBus) ConnInfoUpdate(connInfoUpdate *netsvrProtocol.ConnInfoUpdate) {
	n.sendToSocketByUniqId(connInfoUpdate.GetUniqId(), n.pack(netsvrProtocol.Cmd_ConnInfoUpdate, connInfoUpdate))
}

// ConnInfoDelete 删除连接存储在网关中的信息
func (n *NetBus) ConnInfoDelete(connInfoDelete *netsvrProtocol.ConnInfoDelete) {
	n.sendToSocketByUniqId(connInfoDelete.GetUniqId(), n.pack(netsvrProtocol.Cmd_ConnInfoDelete, connInfoDelete))
}

// TopicSubscribe 令某个连接订阅若干个主题
func (n *NetBus) TopicSubscribe(uniqId string, topics []string, data []byte) {
	req := &netsvrProtocol.TopicSubscribe{
		UniqId: uniqId,
		Topics: topics,
		Data:   data,
	}
	n.sendToSocketByUniqId(uniqId, n.pack(netsvrProtocol.Cmd_TopicSubscribe, req))
}

// TopicUnsubscribe 令某个连接取消订阅若干个主题
func (n *NetBus) TopicUnsubscribe(uniqId string, topics []string, data []byte) {
	req := &netsvrProtocol.TopicUnsubscribe{
		UniqId: uniqId,
		Topics: topics,
		Data:   data,
	}
	n.sendToSocketByUniqId(uniqId, n.pack(netsvrProtocol.Cmd_TopicUnsubscribe, req))
}

// TopicDelete 删除网关中的若干个主题
func (n *NetBus) TopicDelete(topics []string, data []byte) {
	req := &netsvrProtocol.TopicDelete{
		Topics: topics,
		Data:   data,
	}
	n.sendToSockets(n.pack(netsvrProtocol.Cmd_TopicDelete, req))
}

// ============================== 发送：强制下线 ==============================

// ForceOffline 强制关闭某几个连接
func (n *NetBus) ForceOffline(uniqIds []string, data []byte) {
	if len(uniqIds) == 0 {
		return
	}
	if n.isSinglePoint() {
		n.sendToSockets(n.pack(netsvrProtocol.Cmd_ForceOffline, &netsvrProtocol.ForceOffline{UniqIds: uniqIds, Data: data}))
		return
	}
	for addrAsHex, currentUniqIds := range n.getUniqIdsGroupByAddrAsHex(uniqIds) {
		n.sendToSocketByAddrAsHex(addrAsHex, n.pack(netsvrProtocol.Cmd_ForceOffline, &netsvrProtocol.ForceOffline{UniqIds: currentUniqIds, Data: data}))
	}
}

// ForceOfflineByCustomerId 强制关闭某几个客户的所有连接
func (n *NetBus) ForceOfflineByCustomerId(customerIds []string, data []byte) {
	if len(customerIds) == 0 {
		return
	}
	//因为不知道客户id在哪个网关，所以给所有网关发送
	n.sendToSockets(n.pack(netsvrProtocol.Cmd_ForceOfflineByCustomerId, &netsvrProtocol.ForceOfflineByCustomerId{CustomerIds: customerIds, Data: data}))
}

// ForceOfflineGuest 强制关闭某几个空session、空customerId的连接
func (n *NetBus) ForceOfflineGuest(uniqIds []string, data []byte, delay int32) {
	if len(uniqIds) == 0 {
		return
	}
	if n.isSinglePoint() {
		n.sendToSockets(n.pack(netsvrProtocol.Cmd_ForceOfflineGuest, &netsvrProtocol.ForceOfflineGuest{UniqIds: uniqIds, Delay: delay, Data: data}))
		return
	}
	for addrAsHex, currentUniqIds := range n.getUniqIdsGroupByAddrAsHex(uniqIds) {
		n.sendToSocketByAddrAsHex(addrAsHex, n.pack(netsvrProtocol.Cmd_ForceOfflineGuest, &netsvrProtocol.ForceOfflineGuest{UniqIds: currentUniqIds, Delay: delay, Data: data}))
	}
}

// ============================== 查询 ==============================

// CheckOnline 检查某几个uniqId是否在线
func (n *NetBus) CheckOnline(uniqIds []string) *ret.CheckOnlineRet {
	res := ret.CheckOnlineRet{Data: make(map[string]*netsvrProtocol.CheckOnlineResp)}
	if len(uniqIds) == 0 {
		return &res
	}
	if n.isSinglePoint() {
		socket := n.getTaskSocketByUniqId(uniqIds[0])
		if socket == nil {
			return &res
		}
		defer socket.Release()
		socket.Send(n.pack(netsvrProtocol.Cmd_CheckOnline, &netsvrProtocol.CheckOnlineReq{UniqIds: uniqIds}))
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::CheckOnline failed because the connection to the netsvr was disconnected")
			return &res
		}
		resp := &netsvrProtocol.CheckOnlineResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.CheckOnlineResp failed", "error", err)
			return &res
		}
		res.Data[socket.GetAddr()] = resp
		return &res
	}
	for addrAsHex, currentUniqIds := range n.getUniqIdsGroupByAddrAsHex(uniqIds) {
		socket := n.taskSocketPoolManger.GetSocket(addrAsHex)
		if socket == nil {
			continue
		}
		func() {
			defer socket.Release()
			socket.Send(n.pack(netsvrProtocol.Cmd_CheckOnline, &netsvrProtocol.CheckOnlineReq{UniqIds: currentUniqIds}))
			respData := socket.Receive()
			if respData == nil {
				log.Error("call Cmd::CheckOnline failed because the connection to the netsvr was disconnected")
				return
			}
			resp := &netsvrProtocol.CheckOnlineResp{}
			if err := proto.Unmarshal(respData[4:], resp); err != nil {
				log.Error("unmarshal netsvrProtocol.CheckOnlineResp failed", "error", err)
				return
			}
			res.Data[socket.GetAddr()] = resp
		}()
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
		resp := &netsvrProtocol.UniqIdListResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.UniqIdListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// UniqIdCount 统计所有网关中存储的uniqId数量
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
		resp := &netsvrProtocol.UniqIdCountResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.UniqIdCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// CustomerIdList 获取所有网关中存储的customerId
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
		resp := &netsvrProtocol.CustomerIdListResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.CustomerIdListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// CustomerIdCount 统计所有网关中存储的customerId数量。
// 注意：各网关数量之和不一定等于总在线客户数，一个客户可能有多个设备连接到不同网关。
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
		resp := &netsvrProtocol.CustomerIdCountResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.CustomerIdCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// TopicList 获取所有网关中存储的主题
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
		resp := &netsvrProtocol.TopicListResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// TopicCount 统计所有网关中存储的主题数量
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
		resp := &netsvrProtocol.TopicCountResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// TopicUniqIdList 获取某几个主题包含的uniqId
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
		resp := &netsvrProtocol.TopicUniqIdListResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicUniqIdListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// TopicUniqIdCount 统计某几个主题包含的连接数（去重统计）。
// topics 为空时统计网关中全部主题。
func (n *NetBus) TopicUniqIdCount(topics []string) *ret.TopicUniqIdCountRet {
	res := ret.TopicUniqIdCountRet{Data: make(map[string]*netsvrProtocol.TopicUniqIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicUniqIdCount, &netsvrProtocol.TopicUniqIdCountReq{Topics: topics})
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::TopicUniqIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		resp := &netsvrProtocol.TopicUniqIdCountResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicUniqIdCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// TopicCustomerIdList 获取某几个主题的customerId
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
		resp := &netsvrProtocol.TopicCustomerIdListResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicCustomerIdListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// TopicCustomerIdCount 统计某几个主题的customerId数量（去重统计）。
// topics 为空时统计网关中全部主题。
func (n *NetBus) TopicCustomerIdCount(topics []string) *ret.TopicCustomerIdCountRet {
	res := ret.TopicCustomerIdCountRet{Data: make(map[string]*netsvrProtocol.TopicCustomerIdCountResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_TopicCustomerIdCount, &netsvrProtocol.TopicCustomerIdCountReq{Topics: topics})
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::TopicCustomerIdCount failed because the connection to the netsvr was disconnected")
			continue
		}
		resp := &netsvrProtocol.TopicCustomerIdCountResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicCustomerIdCountResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// TopicCustomerIdToUniqIdsList 获取某几个主题的customerId以及对应的uniqId列表
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
		resp := &netsvrProtocol.TopicCustomerIdToUniqIdsListResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.TopicCustomerIdToUniqIdsListResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// ConnInfo 获取某几个uniqId的连接信息
func (n *NetBus) ConnInfo(uniqIds []string, reqSession bool, reqCustomerId bool, reqTopic bool) *ret.ConnInfoRet {
	res := ret.ConnInfoRet{Data: make(map[string]*netsvrProtocol.ConnInfoResp)}
	if len(uniqIds) == 0 {
		return &res
	}
	if n.isSinglePoint() {
		socket := n.getTaskSocketByUniqId(uniqIds[0])
		if socket == nil {
			return &res
		}
		defer socket.Release()
		req := &netsvrProtocol.ConnInfoReq{
			UniqIds:       uniqIds,
			ReqSession:    reqSession,
			ReqCustomerId: reqCustomerId,
			ReqTopic:      reqTopic,
		}
		socket.Send(n.pack(netsvrProtocol.Cmd_ConnInfo, req))
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::ConnInfo failed because the connection to the netsvr was disconnected")
			return &res
		}
		resp := &netsvrProtocol.ConnInfoResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.ConnInfoResp failed", "error", err)
			return &res
		}
		res.Data[socket.GetAddr()] = resp
		return &res
	}
	for addrAsHex, currentUniqIds := range n.getUniqIdsGroupByAddrAsHex(uniqIds) {
		socket := n.taskSocketPoolManger.GetSocket(addrAsHex)
		if socket == nil {
			continue
		}
		func() {
			defer socket.Release()
			req := &netsvrProtocol.ConnInfoReq{
				UniqIds:       currentUniqIds,
				ReqSession:    reqSession,
				ReqCustomerId: reqCustomerId,
				ReqTopic:      reqTopic,
			}
			socket.Send(n.pack(netsvrProtocol.Cmd_ConnInfo, req))
			respData := socket.Receive()
			if respData == nil {
				log.Error("call Cmd::ConnInfo failed because the connection to the netsvr was disconnected")
				return
			}
			resp := &netsvrProtocol.ConnInfoResp{}
			if err := proto.Unmarshal(respData[4:], resp); err != nil {
				log.Error("unmarshal netsvrProtocol.ConnInfoResp failed", "error", err)
				return
			}
			res.Data[socket.GetAddr()] = resp
		}()
	}
	return &res
}

// ConnInfoByCustomerId 获取某几个customerId的连接信息
func (n *NetBus) ConnInfoByCustomerId(customerIds []string, reqSession bool, reqUniqId bool, reqTopic bool) *ret.ConnInfoByCustomerIdRet {
	res := ret.ConnInfoByCustomerIdRet{Data: make(map[string]*netsvrProtocol.ConnInfoByCustomerIdResp)}
	taskSockets := n.taskSocketPoolManger.GetSockets()
	defer func() {
		for _, socket := range taskSockets {
			socket.Release()
		}
	}()
	message := n.pack(netsvrProtocol.Cmd_ConnInfoByCustomerId, &netsvrProtocol.ConnInfoByCustomerIdReq{
		CustomerIds: customerIds,
		ReqSession:  reqSession,
		ReqUniqId:   reqUniqId,
		ReqTopic:    reqTopic,
	})
	for _, socket := range taskSockets {
		socket.Send(message)
		respData := socket.Receive()
		if respData == nil {
			log.Error("call Cmd::ConnInfoByCustomerId failed because the connection to the netsvr was disconnected")
			continue
		}
		resp := &netsvrProtocol.ConnInfoByCustomerIdResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.ConnInfoByCustomerIdResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
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
		resp := &netsvrProtocol.MetricsResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.MetricsResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// Limit 设置并返回网关的限流配置。addr 为空表示对全部网关生效。
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
		resp := &netsvrProtocol.LimitResp{}
		if err := proto.Unmarshal(respData[4:], resp); err != nil {
			log.Error("unmarshal netsvrProtocol.LimitResp failed", "error", err)
			continue
		}
		res.Data[socket.GetAddr()] = resp
	}
	return &res
}

// ============================== 内部方法 ==============================

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
