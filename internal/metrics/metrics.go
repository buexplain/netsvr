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

// Package metrics 服务状态统计模块
package metrics

import (
	gMetrics "github.com/rcrowley/go-metrics"
	"netsvr/configs"
)

type Item int

// 支持统计的服务状态
const (
	// ItemCustomerConnOpenCount 统计客户连接的打开次数
	ItemCustomerConnOpenCount Item = iota + 1
	// ItemCustomerConnCloseCount 统计客户连接的关闭次数
	ItemCustomerConnCloseCount
	// ItemCustomerHeartbeatCount 统计客户连接的心跳次数
	ItemCustomerHeartbeatCount
	// ItemCustomerTransferCount 统计客户数据转发到worker的次数
	ItemCustomerTransferCount
	// ItemCustomerTransferByte 统计客户数据转发到worker的字节数
	ItemCustomerTransferByte
	// ItemCustomerWriteCount 统计往客户写入数据成功的次数
	ItemCustomerWriteCount
	// ItemCustomerWriteByte 统计往客户写入数据成功的字节数
	ItemCustomerWriteByte
	// ItemOpenRateLimitCount 统计连接打开的限流次数
	ItemOpenRateLimitCount
	// ItemMessageRateLimitCount 统计客户消息限流次数
	ItemMessageRateLimitCount
	// ItemWorkerToBusinessFailedCount 统计worker到business的失败次数
	ItemWorkerToBusinessFailedCount
	// ItemCustomerWriteFailedCount 统计往客户写入数据失败的次数
	ItemCustomerWriteFailedCount
	// ItemCustomerWriteFailedByte 统计往客户写入数据失败的字节数
	ItemCustomerWriteFailedByte
	//结束符
	itemLen
)

var Registry = make([]*Status, itemLen)

func init() {
	inMetricsItem := func(i int) bool {
		if configs.Config.Metrics.Item == nil {
			return false
		}
		for _, v := range configs.Config.Metrics.Item {
			if v == i {
				return true
			}
		}
		return false
	}
	for item := 0; item < len(Registry); item++ {
		//0号元素是空壳子，不进行统计
		if item == 0 {
			Registry[0] = nil
			continue
		}
		s := Status{Item: Item(item)}
		//判断是否为配置要求进行统计
		if inMetricsItem(item) {
			//配置文件要求统计，则初始化真实地统计对象
			s.Meter = gMetrics.NewMeter()
		} else {
			//配置文件不要求统计，则初始化一个空壳子上去，这个空壳子是个空函数调用，不影响程序性能
			s.Meter = gMetrics.NilMeter{}
		}
		Registry[item] = &s
	}
}
