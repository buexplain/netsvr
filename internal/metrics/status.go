/**
* Copyright 2022 buexplain@qq.com
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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	gMetrics "github.com/rcrowley/go-metrics"
)

type Status struct {
	Item        int
	Meter       gMetrics.Meter
	MeanRateMax gMetrics.Gauge
	Rate1Max    gMetrics.Gauge
	Rate5Max    gMetrics.Gauge
	Rate15Max   gMetrics.Gauge
}

func (r *Status) ToStatusResp() *netsvrProtocol.MetricsStatusResp {
	if _, ok := r.Meter.(gMetrics.NilMeter); ok {
		return nil
	}
	ret := netsvrProtocol.MetricsStatusResp{}
	r.recordMax()
	ret.Count = r.Meter.Count()
	sp := r.Meter.Snapshot()
	ret.MeanRate = float32(sp.RateMean())
	ret.MeanRateMax = float32(r.MeanRateMax.Value())
	ret.Rate1 = float32(sp.Rate1())
	ret.Rate1Max = float32(r.Rate1Max.Value())
	ret.Rate5 = float32(sp.Rate5())
	ret.Rate5Max = float32(r.Rate5Max.Value())
	ret.Rate15 = float32(sp.Rate15())
	ret.Rate15Max = float32(r.Rate15Max.Value())
	return &ret
}

func (r *Status) recordMax() {
	var a, b int64
	sp := r.Meter.Snapshot()
	a = int64(sp.RateMean())
	b = r.MeanRateMax.Value()
	if b < a {
		r.MeanRateMax.Update(a)
	}
	a = int64(sp.Rate1())
	b = r.Rate1Max.Value()
	if b < a {
		r.Rate1Max.Update(a)
	}
	a = int64(sp.Rate5())
	b = r.Rate5Max.Value()
	if b < a {
		r.Rate5Max.Update(a)
	}
	a = int64(sp.Rate15())
	b = r.Rate15Max.Value()
	if b < a {
		r.Rate15Max.Update(a)
	}
}
