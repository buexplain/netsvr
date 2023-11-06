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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v2/netsvr"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/userDb"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
)

type singleCastBulk struct{}

var SingleCastBulk = singleCastBulk{}

func (r singleCastBulk) Init(processor *connProcessor.ConnProcessor) {
	processor.RegisterBusinessCmd(protocol.RouterSingleCastBulkForUniqId, r.BulkForUniqId)
}

// SingleCastBulkForUniqIdParam 客户端发送的单播信息
type SingleCastBulkForUniqIdParam struct {
	Message []string
	UniqIds []string `json:"uniqIds"`
}

// BulkForUniqId 批量单播给某几个uniqId
func (singleCastBulk) BulkForUniqId(tf *netsvrProtocol.Transfer, param string, processor *connProcessor.ConnProcessor) {
	payload := SingleCastBulkForUniqIdParam{}
	if err := json.Unmarshal(testUtils.StrToReadOnlyBytes(param), &payload); err != nil {
		log.Logger.Error().Err(err).Str("param", param).Msg("Parse SingleCastBulkForUniqIdParam failed")
		return
	}
	var fromUser string
	currentUser := userDb.ParseNetSvrInfo(tf.Session)
	if currentUser == nil {
		fromUser = fmt.Sprintf("uniqId(%s)", tf.UniqId)
	} else {
		fromUser = currentUser.Name
	}
	//构建批量单播数据
	ret := &netsvrProtocol.SingleCastBulk{UniqIds: make([]string, 0, len(payload.UniqIds)), Data: make([][]byte, 0, len(payload.UniqIds))}
	for _, data := range payload.Message {
		msg := map[string]interface{}{"fromUser": fromUser, "message": data}
		ret.Data = append(ret.Data, testUtils.NewResponse(protocol.RouterSingleCastBulkForUniqId, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg}))
	}
	ret.UniqIds = payload.UniqIds
	//发到网关
	processor.Send(ret, netsvrProtocol.Cmd_SingleCastBulk)
}
