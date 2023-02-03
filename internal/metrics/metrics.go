package metrics

import (
	gMetrics "github.com/rcrowley/go-metrics"
	"netsvr/configs"
	"time"
)

// 支持统计的服务状态
const (
	ItemCustomerConnOpen  = iota //统计客户连接的打开情况
	ItemCustomerConnClose        //统计客户连接的关闭情况
	ItemCustomerHeartbeat        //统计客户连接的心跳情况
	ItemCustomerTransfer         //统计客户数据转发到worker的情况
	itemLen                      //这个放最后面，用来初始化registry用户
)

var itemName = map[int]string{
	ItemCustomerConnOpen:  "CustomerConnOpen",
	ItemCustomerConnClose: "CustomerConnClose",
	ItemCustomerHeartbeat: "CustomerHeartbeat",
	ItemCustomerTransfer:  "CustomerTransfer",
}

var Registry = make([]*Status, itemLen)

func init() {
	//初始化所有要统计的服务状态
	ok := false
	inMetricsItem := func(i int) bool {
		if configs.Config.MetricsItem == nil {
			return false
		}
		for _, v := range configs.Config.MetricsItem {
			if v == i {
				return true
			}
		}
		return false
	}
	for item := 0; item < itemLen; item++ {
		s := Status{Name: itemName[item]}
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
			//配置不要求统计，则初始化虚拟的
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
	//每隔30秒记录一次最大值
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer func() {
			ticker.Stop()
		}()
		for range ticker.C {
			for _, v := range Registry {
				v.RecordMax()
			}
		}
	}()
}
