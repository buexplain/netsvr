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

package metrics

import (
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	gMetrics "github.com/rcrowley/go-metrics"
)

type Status struct {
	//统计项
	Item  Item
	Meter gMetrics.Meter
}

func (r *Status) ToStatusResp() *netsvrProtocol.MetricsRespItem {
	if _, ok := r.Meter.(gMetrics.NilMeter); ok {
		return nil
	}
	ret := netsvrProtocol.MetricsRespItem{}
	ret.Item = int32(r.Item)
	sp := r.Meter.Snapshot()
	ret.Count = sp.Count()
	ret.MeanRate = float32(sp.RateMean())
	ret.Rate1 = float32(sp.Rate1())
	ret.Rate5 = float32(sp.Rate5())
	ret.Rate15 = float32(sp.Rate15())
	return &ret
}
