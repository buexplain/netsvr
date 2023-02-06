package metrics

import (
	gMetrics "github.com/rcrowley/go-metrics"
	"netsvr/internal/protocol"
)

type Status struct {
	Name        string
	Meter       gMetrics.Meter
	MeanRateMax gMetrics.Gauge
	Rate1Max    gMetrics.Gauge
	Rate5Max    gMetrics.Gauge
	Rate15Max   gMetrics.Gauge
}

func (r *Status) ToStatusResp() *protocol.MetricsStatusResp {
	if _, ok := r.Meter.(gMetrics.NilMeter); ok {
		return nil
	}
	ret := protocol.MetricsStatusResp{}
	r.recordMax()
	ret.Count = r.Meter.Count()
	ret.MeanRate = float32(r.Meter.RateMean())
	ret.MeanRateMax = float32(r.MeanRateMax.Value())
	ret.Rate1 = float32(r.Meter.Rate1())
	ret.Rate1Max = float32(r.Rate1Max.Value())
	ret.Rate5 = float32(r.Meter.Rate5())
	ret.Rate5Max = float32(r.Rate5Max.Value())
	ret.Rate15 = float32(r.Meter.Rate15())
	ret.Rate15Max = float32(r.Rate15Max.Value())
	return &ret
}

func (r *Status) recordMax() {
	var a, b int64
	a = int64(r.Meter.RateMean())
	b = r.MeanRateMax.Value()
	if b < a {
		r.MeanRateMax.Update(a)
	}
	a = int64(r.Meter.Rate1())
	b = r.Rate1Max.Value()
	if b < a {
		r.Rate1Max.Update(a)
	}
	a = int64(r.Meter.Rate5())
	b = r.Rate5Max.Value()
	if b < a {
		r.Rate5Max.Update(a)
	}
	a = int64(r.Meter.Rate15())
	b = r.Rate15Max.Value()
	if b < a {
		r.Rate15Max.Update(a)
	}
}
