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
	"time"
)

// 支持统计的服务状态
const (
	ItemCustomerConnOpen       = iota //统计客户连接的打开情况
	ItemCustomerConnClose             //统计客户连接的关闭情况
	ItemCustomerHeartbeat             //统计客户连接的心跳情况
	ItemCustomerTransferNumber        //统计客户数据转发到worker的次数情况
	ItemCustomerTransferByte          //统计客户数据转发到worker的字节数情况
	itemLen                           //结束符
)

var Registry = make([]*Status, itemLen)

func init() {
	//初始化所有要统计的服务状态
	ok := false
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
		s := Status{Item: item}
		//判断是否为配置要求进行统计
		if inMetricsItem(item) {
			ok = true
			//配置要求统计，则初始化真实的
			s.Meter = gMetrics.NewMeter()
			s.MeanRateMax = gMetrics.NewGauge()
			s.Rate1Max = gMetrics.NewGauge()
			s.Rate5Max = gMetrics.NewGauge()
			s.Rate15Max = gMetrics.NewGauge()
		} else {
			//配置不要求统计，则初始化一个空壳子上去
			s.Meter = gMetrics.NilMeter{}
			s.MeanRateMax = gMetrics.NilGauge{}
			s.Rate1Max = gMetrics.NilGauge{}
			s.Rate5Max = gMetrics.NilGauge{}
			s.Rate15Max = gMetrics.NilGauge{}
		}
		Registry[item] = &s
	}
	//如果没有初始化任何统计服务，则不开启记录最大值的协程
	if !ok {
		return
	}
	//每隔n秒记录一次最大值，这个最大值并不精确
	go func() {
		ticker := time.NewTicker(configs.Config.Metrics.MaxRecordInterval)
		defer func() {
			ticker.Stop()
		}()
		for range ticker.C {
			for _, v := range Registry {
				v.recordMax()
			}
		}
	}()
}
