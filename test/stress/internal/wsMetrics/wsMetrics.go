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

type WsStatusSnapshot struct {
	//花费的时间
	SpendTime time.Duration
	//发送的消息字节数
	Send int64
	//接收的消息字节数
	Receive int64
}

type WsStatus struct {
	//模块名字
	Name string
	//第几阶段
	Step int
	//开始时间
	StartTime time.Time
	//全部连接构建完毕的数据快照
	ConnectOK WsStatusSnapshot
	//总数据，扣除全部连接构建完毕的数据后的数据
	ConnectRunning WsStatusSnapshot
	//总的在线连接数
	Online gMetrics.Counter
	//总发送的消息字节数
	Send gMetrics.Counter
	//总接收的消息字节数
	Receive gMetrics.Counter
}

func (r *WsStatus) RecordConnectOK() {
	r.ConnectOK.SpendTime = time.Now().Sub(r.StartTime)
	r.ConnectOK.Send = r.Send.Count()
	r.ConnectOK.Receive = r.Receive.Count()
}

func (r *WsStatus) RecordConnectRunning() {
	r.ConnectRunning.SpendTime = time.Now().Sub(r.StartTime) - r.ConnectOK.SpendTime
	r.ConnectRunning.Send = r.Send.Count() - r.ConnectOK.Send
	r.ConnectRunning.Receive = r.Receive.Count() - r.ConnectOK.Receive
}

func (r *WsStatus) ToTableRow() map[string]string {
	currentTime := time.Now()
	ret := map[string]string{}
	ret["模块"] = r.Name
	ret["阶段"] = fmt.Sprintf("%d", r.Step)
	ret["连接数"] = fmt.Sprintf("%d", r.Online.Count())
	ret["构建中耗时 "] = r.ConnectOK.SpendTime.String()
	ret["构建中发送"] = bytesToNice(r.ConnectOK.Send)
	ret["构建中接收"] = bytesToNice(r.ConnectOK.Receive)
	ret["构建后耗时"] = (currentTime.Sub(r.StartTime) - r.ConnectOK.SpendTime).String()
	ret["构建后发送"] = bytesToNice(r.Send.Count() - r.ConnectOK.Send)
	ret["构建后接收"] = bytesToNice(r.Receive.Count() - r.ConnectOK.Receive)
	ret["总耗时"] = currentTime.Sub(r.StartTime).String()
	ret["总发送"] = bytesToNice(r.Send.Count())
	ret["总接收"] = bytesToNice(r.Receive.Count())
	return ret
}

func (r *WsStatus) ToTotal(total *WsStatus) {
	//模块名字
	total.Name = r.Name
	//连接数
	total.Online.Inc(r.Online.Count())
	//连接构建期间
	total.ConnectOK.SpendTime += r.ConnectOK.SpendTime
	total.ConnectOK.Send += r.ConnectOK.Send
	total.ConnectOK.Receive += r.ConnectOK.Receive
	//连接构建完毕到结束时
	total.ConnectRunning.SpendTime += r.ConnectRunning.SpendTime
	total.ConnectRunning.Send += r.ConnectRunning.Send
	total.ConnectRunning.Receive += r.ConnectRunning.Receive
	//总数据
	total.Send.Inc(r.Send.Count())
	total.Receive.Inc(r.Receive.Count())
	if r.StartTime.Compare(total.StartTime) == -1 {
		total.StartTime = r.StartTime
	}
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
	if tmp.Name != "" && tmp.Step != -1 {
		Collect.add(tmp)
	}
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

func bytesToNice(x int64) string {
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	intPartIndex := 0
	tmp := x
	for {
		tmp /= 1024
		if tmp < 1 {
			break
		}
		intPartIndex++
	}
	intPart := math.Pow(1024, float64(intPartIndex))
	return fmt.Sprintf("%.3f %s", float64(x)/intPart, sizes[intPartIndex])
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
	header := []string{"模块", "阶段", "连接数", "构建中耗时 ", "构建中发送", "构建中接收", "构建后耗时", "构建后发送", "构建后接收", "总耗时", "总发送", "总接收"}
	table.SetHeader(header)
	total := New("", -1)
	for _, m := range moduleSlice {
		statusSlice := r.c[m]
		subtotal := New("", 0)
		for _, status := range statusSlice {
			status.RecordConnectRunning()
			//记录总数
			status.ToTotal(total)
			//记录本模块数
			status.ToTotal(subtotal)
			//写入当前步骤数据
			tmp := status.ToTableRow()
			row := make([]string, 0, len(header))
			for _, v := range header {
				row = append(row, tmp[v])
			}
			table.Append(row)
		}
		tmp := subtotal.ToTableRow()
		tmp["阶段"] = "小计"
		tmp["构建后耗时"] = "-" //因为有时间重叠，所以不能做相加计算
		tmp["总耗时"] = "-"
		row := make([]string, 0, len(header))
		for _, v := range header {
			row = append(row, tmp[v])
		}
		table.Append(row)
	}
	tmp := total.ToTableRow()
	tmp["模块"] = "总计"
	tmp["阶段"] = "-"
	tmp["构建后耗时"] = "-"
	tmp["总耗时"] = "-"
	row := make([]string, 0, len(header))
	for _, v := range header {
		row = append(row, tmp[v])
	}
	table.Append(row)
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
