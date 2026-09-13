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
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"

	"github.com/buexplain/netsvr-protocol-go/v7/netsvrProtocol"
)

type singleCastBulk struct{}

var SingleCastBulk = singleCastBulk{}

func init() {
	businessCmdCallback[protocol.RouterSingleCastBulk] = SingleCastBulk.UniqId
	businessCmdCallback[protocol.RouterSingleCastBulkByCustomerId] = SingleCastBulk.CustomerId
}

// SingleCastBulkParam 客户端发送的单播信息
type SingleCastBulkParam struct {
	Message []string
	UniqIds []string `json:"uniqIds"`
}

// UniqId 批量单播给某几个uniqId
func (singleCastBulk) UniqId(tf *netsvrProtocol.Transfer, param string) {
	payload := SingleCastBulkParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse SingleCastBulkParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	//按 uniqId 归集数据后构建批量单播的 items
	itemData := make(map[string][][]byte)
	var itemOrder []string
	for index, data := range payload.Message {
		//单个uniqId时，所有消息都发给该uniqId；否则按顺序一一对应
		var uniqId string
		if len(payload.UniqIds) == 1 {
			uniqId = payload.UniqIds[0]
		} else if index < len(payload.UniqIds) {
			uniqId = payload.UniqIds[index]
		} else {
			break
		}
		if _, ok := itemData[uniqId]; !ok {
			itemOrder = append(itemOrder, uniqId)
		}
		//空消息按空数据下发，网关会跳过零长度的数据项
		if data == "" {
			itemData[uniqId] = append(itemData[uniqId], nil)
			continue
		}
		msg := map[string]interface{}{"fromUser": fromUser, "message": data}
		itemData[uniqId] = append(itemData[uniqId], testUtils.NewResponse(protocol.RouterSingleCastBulk, map[string]interface{}{
			"code":    0,
			"message": "收到一条信息",
			"data":    msg,
		}))
	}
	items := make([]*netsvrProtocol.SingleCastBulkItem, 0, len(itemOrder))
	for _, uniqId := range itemOrder {
		items = append(items, &netsvrProtocol.SingleCastBulkItem{UniqIds: []string{uniqId}, Data: itemData[uniqId]})
	}
	//发到网关
	netBus.NetBus.SingleCastBulk(items)
}

// SingleCastBulkByCustomerIdParam 客户端发送的单播信息
type SingleCastBulkByCustomerIdParam struct {
	Message     []string
	CustomerIds []string `json:"customerIds"`
}

// CustomerId 批量单播给某几个customerId
func (singleCastBulk) CustomerId(tf *netsvrProtocol.Transfer, param string) {
	payload := SingleCastBulkByCustomerIdParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse SingleCastBulkByCustomerIdParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	//按 customerId 归集数据后构建批量单播的 items
	itemData := make(map[string][][]byte)
	var itemOrder []string
	for index, data := range payload.Message {
		//单个customerId时，所有消息都发给该customerId；否则按顺序一一对应
		var customerId string
		if len(payload.CustomerIds) == 1 {
			customerId = payload.CustomerIds[0]
		} else if index < len(payload.CustomerIds) {
			customerId = payload.CustomerIds[index]
		} else {
			break
		}
		if _, ok := itemData[customerId]; !ok {
			itemOrder = append(itemOrder, customerId)
		}
		//空消息按空数据下发，网关会跳过零长度的数据项
		if data == "" {
			itemData[customerId] = append(itemData[customerId], nil)
			continue
		}
		msg := map[string]interface{}{"fromUser": fromUser, "message": data}
		itemData[customerId] = append(itemData[customerId], testUtils.NewResponse(protocol.RouterSingleCastBulkByCustomerId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg}))
	}
	items := make([]*netsvrProtocol.SingleCastBulkByCustomerIdItem, 0, len(itemOrder))
	for _, customerId := range itemOrder {
		items = append(items, &netsvrProtocol.SingleCastBulkByCustomerIdItem{CustomerIds: []string{customerId}, Data: itemData[customerId]})
	}
	//发到网关
	netBus.NetBus.SingleCastBulkByCustomerId(items)
}
