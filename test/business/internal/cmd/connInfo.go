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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
	"netsvr/test/pkg/utils/netSvrPool"
)

type connInfo struct{}

var ConnInfo = connInfo{}

func (r connInfo) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterConnInfo, r.RequestConnInfo)
	processor.RegisterBusinessCmd(protocol.RouterConnInfoByCustomerId, r.RequestConnInfoByCustomerId)
}

// RequestConnInfo 获取我的连接信息
func (connInfo) RequestConnInfo(tf *netsvrProtocol.Transfer, _ string, processor *connProcessor.ConnProcessor) {
	req := &netsvrProtocol.ConnInfoReq{}
	req.ReqTopic = true
	req.ReqSession = true
	req.ReqCustomerId = true
	req.UniqIds = []string{tf.UniqId}
	resp := &netsvrProtocol.ConnInfoResp{}
	netSvrPool.Request(req, netsvrProtocol.Cmd_ConnInfo, resp)
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{"connInfo": resp.Items[tf.UniqId]}
	ret.Data = testUtils.NewResponse(protocol.RouterConnInfo, map[string]interface{}{"code": 0, "message": "获取我的连接信息成功", "data": msg})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}

// ConnInfoByCustomerIdParam 强制踢下线某个用户
type ConnInfoByCustomerIdParam struct {
	CustomerIds []string `json:"customerIds"`
}

// RequestConnInfoByCustomerId 获取customerId的连接信息
func (connInfo) RequestConnInfoByCustomerId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := new(ConnInfoByCustomerIdParam)
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse ConnInfoByCustomerIdParam failed")
		return
	}
	req := &netsvrProtocol.ConnInfoByCustomerIdReq{}
	req.ReqTopic = true
	req.ReqSession = true
	req.ReqUniqId = true
	req.CustomerIds = payload.CustomerIds
	resp := &netsvrProtocol.ConnInfoByCustomerIdResp{}
	netSvrPool.Request(req, netsvrProtocol.Cmd_ConnInfoByCustomerId, resp)
	ret := &netsvrProtocol.SingleCast{}
	ret.UniqId = tf.UniqId
	msg := map[string]interface{}{"connInfo": resp.Items}
	ret.Data = testUtils.NewResponse(protocol.RouterConnInfoByCustomerId, map[string]interface{}{"code": 0, "message": "获取customerId的连接信息成功", "data": msg})
	processor.Send(ret, netsvrProtocol.Cmd_SingleCast)
}
