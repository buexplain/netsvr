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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v3/netsvr"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/binder"
	"netsvr/internal/customer/info"
	customerManager "netsvr/internal/customer/manager"
	"netsvr/internal/log"
	workerManager "netsvr/internal/worker/manager"
)

// ConnInfoByCustomerId 获取customerId的连接信息
func ConnInfoByCustomerId(param []byte, processor *workerManager.ConnProcessor) {
	payload := &netsvrProtocol.ConnInfoByCustomerIdReq{}
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.ConnInfoByCustomerIdReq failed")
		return
	}
	// 获取customerId对应的uniqId列表
	customerIdUniqIds := binder.Binder.GetUniqIdsByCustomerIds(payload.CustomerIds)
	ret := &netsvrProtocol.ConnInfoByCustomerIdResp{Items: map[string]*netsvrProtocol.ConnInfoByCustomerIdRespItems{}}
	for customerId, uniqIds := range customerIdUniqIds {
		items := netsvrProtocol.ConnInfoByCustomerIdRespItems{}
		// 遍历uniqId列表，获取连接信息
		for _, uniqId := range uniqIds {
			conn := customerManager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			session, ok := conn.SessionWithLock().(*info.Info)
			if !ok {
				continue
			}
			item := &netsvrProtocol.ConnInfoByCustomerIdRespItem{}
			session.GetConnInfoByCustomerIdOnSafe(payload, item)
			items.Items = append(items.Items, item)
		}
		// 如果items不为空，则添加到返回值中
		if len(items.Items) > 0 {
			ret.Items[customerId] = &items
		}
	}
	processor.Send(ret, netsvrProtocol.Cmd_ConnInfoByCustomerId)
}
