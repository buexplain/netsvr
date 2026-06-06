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
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/pkg/protocol"
	testUtils "netsvr/test/pkg/utils"
	"sort"
)

type metrics struct{}

var Metrics = metrics{}

func init() {
	businessCmdCallback[protocol.RouterMetrics] = Metrics.Request
}

type Item struct {
	sortBy      int32
	Description string  `json:"description"`
	Count       int64   `json:"count"`
	MeanRate    float32 `json:"meanRate"`
	Rate1       float32 `json:"rate1"`
	Rate5       float32 `json:"rate5"`
	Rate15      float32 `json:"rate15"`
}

var description = map[int]string{
	1:  "客户连接的打开次数",
	2:  "客户连接的关闭次数",
	3:  "客户连接的心跳次数",
	4:  "客户数据通过worker转发到业务侧的次数",
	5:  "客户数据通过worker转发到业务侧的字节数",
	6:  "往客户写入数据成功次数",
	7:  "往客户写入数据成功字节数",
	8:  "连接打开的限流次数",
	9:  "客户消息限流次数",
	10: "客户数据通过worker转发到业务侧的失败次数",
	11: "往客户写入数据失败次数",
	12: "往客户写入数据失败字节数",
	13: "连接消息限流次数",
	14: "客户数据通过redis队列转发到业务侧的次数",
	15: "客户数据通过redis队列转发到业务侧的字节数",
	16: "客户数据通过redis队列转发到业务侧的失败次数",
	17: "客户数据通过http回调转发到业务侧的次数",
	18: "客户数据通过http回调转发到业务侧的字节数",
	19: "客户数据通过http回调转发到业务侧的失败次数",
	20: "统计客户数据通过amqp091队列转发到业务侧的次数",
	21: "统计客户数据通过amqp091队列转发到业务侧的字节数",
	22: "统计客户数据通过amqp091队列转发到业务侧的失败次数",
}

// Request 获取网关的服务状态
func (metrics) Request(tf *netsvrProtocol.Transfer, _ string) {
	resp := netBus.NetBus.Metrics()
	var data []Item
	for _, metricsResp := range resp.Data {
		for i, item := range metricsResp.Items {
			data = append(data, Item{
				sortBy:      i,
				Description: description[int(i)],
				Count:       item.Count,
				MeanRate:    item.MeanRate,
				Rate1:       item.Rate1,
				Rate5:       item.Rate5,
				Rate15:      item.Rate15,
			})
		}
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].sortBy < data[j].sortBy
	})
	netBus.NetBus.SingleCast(tf.UniqId, testUtils.NewResponse(protocol.RouterMetrics, map[string]interface{}{"code": 0, "message": "获取网关状态的信息成功", "data": data}))
}
