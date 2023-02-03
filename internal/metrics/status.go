package metrics

import gMetrics "github.com/rcrowley/go-metrics"

type Status struct {
	Name        string
	Meter       gMetrics.Meter
	MeanRateMax gMetrics.Gauge
	Rate1Max    gMetrics.Gauge
	Rate5Max    gMetrics.Gauge
	Rate15Max   gMetrics.Gauge
}

func (r *Status) ToMap() map[string]int64 {
	ret := map[string]int64{}
	ret["count"] = r.Meter.Count()
	ret["meanRate"] = int64(r.Meter.RateMean())
	ret["meanRateMax"] = r.MeanRateMax.Value()
	ret["rate1"] = int64(r.Meter.Rate1())
	ret["rate1Max"] = r.Rate1Max.Value()
	ret["rate5"] = int64(r.Meter.Rate5())
	ret["rate5Max"] = r.Rate5Max.Value()
	ret["rate15"] = int64(r.Meter.Rate15())
	ret["rate15Max"] = r.Rate15Max.Value()
	return ret
}

func (r *Status) RecordMax() {
	var a, b int64
	a = int64(r.Meter.RateMean())
	b = r.MeanRateMax.Value()
	if b < a {
		r.MeanRateMax.Update(b)
	}
	a = int64(r.Meter.Rate1())
	b = r.Rate1Max.Value()
	if b < a {
		r.Rate1Max.Update(b)
	}
	a = int64(r.Meter.Rate5())
	b = r.Rate5Max.Value()
	if b < a {
		r.Rate5Max.Update(b)
	}
	a = int64(r.Meter.Rate15())
	b = r.Rate15Max.Value()
	if b < a {
		r.Rate15Max.Update(b)
	}
}
