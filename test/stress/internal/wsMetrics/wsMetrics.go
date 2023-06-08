package wsMetrics

import (
	"bytes"
	"fmt"
	"github.com/olekukonko/tablewriter"
	gMetrics "github.com/rcrowley/go-metrics"
	"math"
	"sort"
	"sync"
	"time"
)

type WsStatus struct {
	//模块名字
	Name string
	//第几阶段
	Step int
	//开始时间
	StartTime time.Time
	//在线人数
	Online gMetrics.Counter
	//发送的消息字节数
	Send gMetrics.Counter
	//接收的消息字节数
	Receive gMetrics.Counter
}

func New(name string, step int) *WsStatus {
	tmp := &WsStatus{
		Name:      name,
		Step:      step,
		StartTime: time.Now(),
		Online:    gMetrics.NewCounter(),
		Send:      gMetrics.NewCounter(),
		Receive:   gMetrics.NewCounter(),
	}
	Collect.add(tmp)
	return tmp
}

type collect struct {
	c   map[string][]*WsStatus
	mux *sync.Mutex
}

func (r *collect) add(status *WsStatus) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if s, ok := r.c[status.Name]; ok {
		r.c[status.Name] = append(s, status)
		return
	}
	r.c[status.Name] = []*WsStatus{status}
}

func bytesToNice(s int64) string {
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	e := math.Floor(math.Log(float64(s)) / math.Log(1024))
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(1024, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}
	return fmt.Sprintf(f, val, suffix)
}

func (r *collect) CountByName(name string) int64 {
	r.mux.Lock()
	defer r.mux.Unlock()
	var targetStatusSlice []*WsStatus
	for _, statusSlice := range r.c {
		for _, status := range statusSlice {
			if status.Name == name {
				targetStatusSlice = statusSlice
				break
			}
		}
	}
	if targetStatusSlice == nil {
		return 0
	}
	var ret int64
	for _, status := range targetStatusSlice {
		ret += status.Online.Count()
	}
	return ret
}

func (r *collect) Count() int64 {
	r.mux.Lock()
	defer r.mux.Unlock()
	var ret int64
	for _, statusSlice := range r.c {
		for _, status := range statusSlice {
			ret += status.Online.Count()
		}
	}
	return ret
}

func (r *collect) ToTable() *bytes.Buffer {
	r.mux.Lock()
	defer r.mux.Unlock()
	var moduleSlice []string
	for _, statusSlice := range r.c {
		for _, status := range statusSlice {
			moduleSlice = append(moduleSlice, status.Name)
			break
		}
	}
	sort.Strings(moduleSlice)
	ret := &bytes.Buffer{}
	table := tablewriter.NewWriter(ret)
	table.SetHeader([]string{"模块", "阶段", "连接数", "发送字节", "接收字节", "持续时间"})
	currentTime := time.Now()
	var totalOnline int64
	var totalSend int64
	var totalReceive int64
	totalStartTime := currentTime
	for _, m := range moduleSlice {
		statusSlice := r.c[m]
		var moduleName string
		var moduleOnline int64
		var moduleSend int64
		var moduleReceive int64
		moduleStartTime := currentTime
		for _, status := range statusSlice {
			//记录总数
			totalOnline += status.Online.Count()
			totalSend += status.Send.Count()
			totalReceive += status.Receive.Count()
			if status.StartTime.Compare(totalStartTime) == -1 {
				totalStartTime = status.StartTime
			}
			//记录本模块数
			moduleName = status.Name
			moduleOnline += status.Online.Count()
			moduleSend += status.Send.Count()
			moduleReceive += status.Receive.Count()
			if status.StartTime.Compare(moduleStartTime) == -1 {
				moduleStartTime = status.StartTime
			}
			table.Append([]string{
				status.Name,
				fmt.Sprintf("%d", status.Step),
				fmt.Sprintf("%d", status.Online.Count()),
				bytesToNice(status.Send.Count()),
				bytesToNice(status.Receive.Count()),
				fmt.Sprintf("%s", currentTime.Sub(status.StartTime).String()),
			})
		}
		table.Append([]string{
			moduleName,
			"小计",
			fmt.Sprintf("%d", moduleOnline),
			bytesToNice(moduleSend),
			bytesToNice(moduleReceive),
			fmt.Sprintf("%s", currentTime.Sub(moduleStartTime).String()),
		})
	}
	table.Append([]string{
		"总计",
		"-",
		fmt.Sprintf("%d", totalOnline),
		bytesToNice(totalSend),
		bytesToNice(totalReceive),
		fmt.Sprintf("%s", currentTime.Sub(totalStartTime).String()),
	})
	table.SetAutoMergeCellsByColumnIndex([]int{0})
	table.SetRowLine(true)
	table.Render()
	return ret
}

var Collect *collect

func init() {
	Collect = &collect{
		c:   map[string][]*WsStatus{},
		mux: &sync.Mutex{},
	}
}
