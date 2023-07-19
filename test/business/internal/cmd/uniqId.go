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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"google.golang.org/protobuf/proto"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type uniqId struct{}

var UniqId = uniqId{}

func (r uniqId) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterUniqIdList, r.RequestList)

	processor.RegisterBusinessCmd(protocol.RouterUniqIdCount, r.RequestCount)
}

// RequestList 获取网关所有的uniqId
func (uniqId) RequestList(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	resp := &netsvrProtocol.UniqIdListResp{}
	testUtils.RequestNetSvr(nil, netsvrProtocol.Cmd_UniqIdList, resp)
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{
		"uniqIds": resp.UniqIds,
	}
	ret.Data = testUtils.NewResponse(protocol.RouterUniqIdList, map[string]interface{}{"code": 0, "message": "获取网关所有的uniqId成功", "data": msg})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}

// RequestCount 获取网关中uniqId的数量
func (uniqId) RequestCount(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	resp := &netsvrProtocol.UniqIdCountResp{}
	testUtils.RequestNetSvr(nil, netsvrProtocol.Cmd_UniqIdCount, resp)
	//将结果单播给客户端
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{
		"count": resp.Count,
	}
	ret.Data = testUtils.NewResponse(protocol.RouterUniqIdCount, map[string]interface{}{"code": 0, "message": "获取网关中uniqId的数量成功", "data": msg})
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_SingleCast
	router.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(router)
	processor.Send(pt)
}
