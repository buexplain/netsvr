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

package gcStats

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed chart.js
var chartJS string

type GCStats struct {
	Timestamp     time.Time
	NumGC         int64
	PauseTotalNs  int64
	PauseNs       []int64
	HeapAlloc     uint64
	HeapSys       uint64
	HeapIdle      uint64
	HeapInuse     uint64
	HeapReleased  uint64
	HeapObjects   uint64
	GCCPUFraction float64
}

type DataPoint struct {
	Time            time.Time `json:"time"`
	NumGC           int64     `json:"num_gc"`
	PauseMs         float64   `json:"pause_ms"`
	AvgPauseMs      float64   `json:"avg_pause_ms"`
	HeapAllocMB     float64   `json:"heap_alloc_mb"`
	HeapInuseMB     float64   `json:"heap_inuse_mb"`
	HeapReleasedMB  float64   `json:"heap_released_mb"`
	HeapUtilization float64   `json:"heap_utilization"`
	HeapObjects     uint64    `json:"heap_objects"`
	GCCPUPercent    float64   `json:"gc_cpu_percent"`
}

var dataPoints []DataPoint

func Start() {
	//127.0.0.1:6065
	if configs.Config.ClientListenAddress == "" {
		return
	}
	//127.0.0.1:6063
	if configs.Config.NetsvrPprofListenAddress == "" {
		return
	}
	checkIsOpen := func(addr string) bool {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			_ = c.Close()
			return true
		}
		var e *net.OpError
		if errors.As(err, &e) && (strings.Contains(e.Err.Error(), "No connection") || strings.Contains(e.Err.Error(), "connection refused")) {
			return false
		}
		return true
	}
	if checkIsOpen(configs.Config.ClientListenAddress) {
		log.Logger.Info().Msg("地址已被占用: " + configs.Config.ClientListenAddress)
		return
	}
	if !checkIsOpen(configs.Config.NetsvrPprofListenAddress) {
		log.Logger.Info().Msg("地址未开启: " + configs.Config.NetsvrPprofListenAddress)
		return
	}
	closed := make(chan struct{})
	go func() {
		select {
		case <-closed:
			return
		case <-time.After(time.Millisecond * 1500):
			break
		}
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			collectAndStore(configs.Config.NetsvrPprofListenAddress)
		}
	}()
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/data", serveData)
	log.Logger.Info().Msgf("点击访问gc监控面板: http://%s", configs.Config.ClientListenAddress)
	if err := http.ListenAndServe(configs.Config.ClientListenAddress, nil); err != nil {
		log.Logger.Error().Msgf("启动失败: %v", err)
		close(closed)
	}
}

func collectAndStore(address string) {
	stats, err := collectMemStats(address)
	if err != nil {
		log.Logger.Error().Msgf("收集 %s 数据失败: %v", address, err)
		return
	}

	var pauseMs float64
	if len(stats.PauseNs) > 0 {
		pauseMs = float64(stats.PauseNs[len(stats.PauseNs)-1]) / float64(time.Millisecond)
	}

	// 计算平均暂停时间
	var avgPauseMs float64
	if stats.NumGC > 0 && stats.PauseTotalNs > 0 {
		avgPauseMs = float64(stats.PauseTotalNs/stats.NumGC) / float64(time.Millisecond)
	}

	// 计算堆利用率
	var heapUtilization float64
	if stats.HeapSys > 0 {
		heapUtilization = float64(stats.HeapInuse) / float64(stats.HeapSys) * 100
	}

	point := DataPoint{
		Time:            stats.Timestamp,
		NumGC:           stats.NumGC,
		PauseMs:         pauseMs,
		AvgPauseMs:      avgPauseMs,
		HeapAllocMB:     float64(stats.HeapAlloc) / 1024 / 1024,
		HeapInuseMB:     float64(stats.HeapInuse) / 1024 / 1024,
		HeapReleasedMB:  float64(stats.HeapReleased) / 1024 / 1024,
		HeapUtilization: heapUtilization,
		HeapObjects:     stats.HeapObjects,
		GCCPUPercent:    stats.GCCPUFraction * 100,
	}

	dataPoints = append(dataPoints, point)
}

func collectMemStats(address string) (*GCStats, error) {
	cmd := exec.Command("gops", "memstats", address)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("执行 gops 命令失败: %w", err)
	}

	stats := &GCStats{
		Timestamp: time.Now(),
	}

	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		parseLine(line, stats)
	}

	return stats, nil
}

func parseLine(line string, stats *GCStats) {
	line = strings.TrimSpace(line)

	switch {
	case strings.HasPrefix(line, "num-gc"):
		if val := extractInt64(line); val != -1 {
			stats.NumGC = val
		}
	case strings.HasPrefix(line, "gc-pause-total"):
		// gc-pause-total 格式可能是 "427.016628ms" 或其他单位
		if val := extractDurationToNs(line); val != -1 {
			stats.PauseTotalNs = val
		}
	case strings.HasPrefix(line, "gc-pause") && !strings.HasPrefix(line, "gc-pause-total") && !strings.HasPrefix(line, "gc-pause-end"):
		// gc-pause: 3407957 (这是纳秒)
		if val := extractInt64(line); val != -1 {
			stats.PauseNs = []int64{val}
		}
	case strings.HasPrefix(line, "heap-alloc"):
		// heap-alloc: 105.98MB (111128904 bytes)
		if val := extractBytesFromParentheses(line); val != ^uint64(0) {
			stats.HeapAlloc = val
		}
	case strings.HasPrefix(line, "heap-sys"):
		if val := extractBytesFromParentheses(line); val != ^uint64(0) {
			stats.HeapSys = val
		}
	case strings.HasPrefix(line, "heap-idle"):
		if val := extractBytesFromParentheses(line); val != ^uint64(0) {
			stats.HeapIdle = val
		}
	case strings.HasPrefix(line, "heap-in-use"):
		if val := extractBytesFromParentheses(line); val != ^uint64(0) {
			stats.HeapInuse = val
		}
	case strings.HasPrefix(line, "heap-released"):
		if val := extractBytesFromParentheses(line); val != ^uint64(0) {
			stats.HeapReleased = val
		}
	case strings.HasPrefix(line, "heap-objects"):
		if val := extractInt64(line); val != -1 {
			stats.HeapObjects = uint64(val)
		}
	case strings.HasPrefix(line, "gc-cpu-fraction"):
		if val := extractFloat64(line); val != -1 {
			stats.GCCPUFraction = val
		}
	}
}

func extractInt64(line string) int64 {
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return -1
	}
	val := strings.TrimSpace(parts[len(parts)-1])
	val = strings.TrimRight(val, " ")
	result, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return -1
	}
	return result
}

func extractFloat64(line string) float64 {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return -1
	}
	val := strings.TrimSpace(parts[1])
	result, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return -1
	}
	return result
}

func extractDurationToNs(line string) int64 {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return -1
	}
	val := strings.TrimSpace(parts[1])

	// 尝试解析带单位的持续时间，如 "427.016628ms"
	if strings.HasSuffix(val, "ms") {
		numStr := strings.TrimSuffix(val, "ms")
		if num, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64); err == nil {
			return int64(num * float64(time.Millisecond))
		}
	} else if strings.HasSuffix(val, "s") && !strings.HasSuffix(val, "ms") {
		numStr := strings.TrimSuffix(val, "s")
		if num, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64); err == nil {
			return int64(num * float64(time.Second))
		}
	} else if strings.HasSuffix(val, "µs") || strings.HasSuffix(val, "us") {
		numStr := strings.TrimSuffix(val, "µs")
		numStr = strings.TrimSuffix(numStr, "us")
		if num, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64); err == nil {
			return int64(num * float64(time.Microsecond))
		}
	}

	// 如果没有单位，尝试直接解析为纳秒
	if num, err := strconv.ParseFloat(val, 64); err == nil {
		return int64(num)
	}

	return -1
}

func extractBytesFromParentheses(line string) uint64 {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return ^uint64(0)
	}
	val := parts[1]

	// 查找括号中的字节数，如 "(111128904 bytes)"
	re := regexp.MustCompile(`\((\d+)\s*bytes\)`)
	matches := re.FindStringSubmatch(val)
	if len(matches) < 2 {
		// 如果没有括号格式，尝试直接解析数字
		re2 := regexp.MustCompile(`[\d.]+`)
		numStr := re2.FindString(val)
		if numStr == "" {
			return ^uint64(0)
		}
		// 这可能是带单位的，如 "105.98MB"
		if strings.Contains(val, "MB") {
			if num, err := strconv.ParseFloat(numStr, 64); err == nil {
				return uint64(num * 1024 * 1024)
			}
		} else if strings.Contains(val, "KB") {
			if num, err := strconv.ParseFloat(numStr, 64); err == nil {
				return uint64(num * 1024)
			}
		} else if strings.Contains(val, "GB") {
			if num, err := strconv.ParseFloat(numStr, 64); err == nil {
				return uint64(num * 1024 * 1024 * 1024)
			}
		}
		return ^uint64(0)
	}

	result, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return ^uint64(0)
	}
	return result
}

func serveHTML(w http.ResponseWriter, _ *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>GC 分析曲线</title>
    <script>` + chartJS + `</script>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        h1 {
            text-align: center;
            color: #333;
        }
        .chart-container {
            background: white;
            padding: 20px;
            margin: 20px 0;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            position: relative;
            cursor: help;
        }
        .chart-container .chart-tooltip {
            visibility: hidden;
            width: 280px;
            background-color: #333;
            color: #fff;
            text-align: left;
            border-radius: 6px;
            padding: 12px 15px;
            position: absolute;
            z-index: 1000;
            bottom: calc(100% + 10px);
            right: 20px;
            opacity: 0;
            transition: opacity 0.2s;
            font-size: 13px;
            line-height: 1.6;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            pointer-events: none;
        }
        .chart-container .chart-tooltip::after {
            content: "";
            position: absolute;
            top: 100%;
            right: 30px;
            border-width: 6px;
            border-style: solid;
            border-color: #333 transparent transparent transparent;
        }
        .chart-container:hover .chart-tooltip {
            visibility: visible;
            opacity: 1;
        }
        .chart-row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
            margin: 20px 0;
        }
        .chart-row .chart-container {
            margin: 0;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 15px;
            margin: 20px 0;
        }
        .stat-card {
            background: white;
            padding: 15px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            text-align: center;
            cursor: help;
            transition: transform 0.2s, box-shadow 0.2s;
            position: relative;
            min-height: 80px;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
        }
        .stat-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 8px rgba(0,0,0,0.15);
        }
        .stat-card .tooltip {
            visibility: hidden;
            width: 240px;
            background-color: #333;
            color: #fff;
            text-align: left;
            border-radius: 6px;
            padding: 10px 12px;
            position: absolute;
            z-index: 1000;
            bottom: calc(100% + 10px);
            left: 50%;
            transform: translateX(-50%);
            opacity: 0;
            transition: opacity 0.2s;
            font-size: 13px;
            line-height: 1.6;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            pointer-events: none;
        }
        .stat-card .tooltip::after {
            content: "";
            position: absolute;
            top: 100%;
            left: 50%;
            transform: translateX(-50%);
            border-width: 6px;
            border-style: solid;
            border-color: #333 transparent transparent transparent;
        }
        .stat-card:hover .tooltip {
            visibility: visible;
            opacity: 1;
        }
        .stat-value {
            font-size: 24px;
            font-weight: bold;
            color: #2196F3;
        }
        .stat-label {
            color: #666;
            margin-top: 5px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>netsvr gc 分析面板</h1>
        
        <div class="stats" id="stats"></div>
        
        <div class="chart-row">
            <div class="chart-container">
                <canvas id="heapChart"></canvas>
                <div class="chart-tooltip">显示堆内存的分配和使用趋势。蓝色线表示已分配的内存，绿色线表示实际使用的内存。两条线的差距反映了内存碎片程度。</div>
            </div>
            
            <div class="chart-container">
                <canvas id="releasedChart"></canvas>
                <div class="chart-tooltip">显示已释放回操作系统的内存量。Go 运行时会定期将空闲内存归还给操作系统，此图帮助了解内存回收效率。</div>
            </div>
        </div>
        
        <div class="chart-row">
            <div class="chart-container">
                <canvas id="utilizationChart"></canvas>
                <div class="chart-tooltip">堆内存利用率 = 使用中内存 / 系统分配内存 × 100%。理想值通常在 50-80% 之间。过低表示内存碎片多，过高可能导致频繁 GC。</div>
            </div>
            
            <div class="chart-container">
                <canvas id="objectsChart"></canvas>
                <div class="chart-tooltip">显示堆中活跃对象的数量变化。如果此曲线持续增长而不下降，可能存在内存泄漏，需要检查代码中的对象引用。</div>
            </div>
        </div>
        
        <div class="chart-row">
            <div class="chart-container">
                <canvas id="pauseChart"></canvas>
                <div class="chart-tooltip">红色实线显示每次 GC 的暂停时间，橙色虚线显示平均暂停时间。Go 的并发 GC 通常能将暂停时间控制在毫秒级别。</div>
            </div>
            
            <div class="chart-container">
                <canvas id="cpuChart"></canvas>
                <div class="chart-tooltip">GC 占用的 CPU 时间百分比。正常情况应低于 5%。如果过高，可能需要调整 GOGC 环境变量或优化内存使用。</div>
            </div>
        </div>
    </div>

    <script>
        let pauseChart, heapChart, cpuChart, utilizationChart, objectsChart, releasedChart;
        let pollingInterval = null;
        let isServerDown = false;
        let statsInitialized = false;
        
        async function fetchData() {
            if (isServerDown) return;
            
            try {
                const response = await fetch('/data');
                if (!response.ok) {
                    throw new Error('服务器响应错误: ' + response.status);
                }
                const data = await response.json();
                updateCharts(data);
                updateStats(data);
            } catch (error) {
                console.error('获取数据失败:', error);
                handleServerDown();
            }
        }

        function handleServerDown() {
            if (isServerDown) return;
            
            isServerDown = true;
            
            // 停止轮询
            if (pollingInterval) {
                clearInterval(pollingInterval);
                pollingInterval = null;
            }
            
            // 显示错误提示
            showServerDownMessage();
        }

        function showServerDownMessage() {
            const container = document.querySelector('.container');
            const existingMsg = document.getElementById('server-down-msg');
            if (existingMsg) return;
            
            const msgDiv = document.createElement('div');
            msgDiv.id = 'server-down-msg';
            msgDiv.style.cssText = 
                'position: fixed;' +
                'top: 20px;' +
                'right: 20px;' +
                'background: #f44336;' +
                'color: white;' +
                'padding: 15px 20px;' +
                'border-radius: 8px;' +
                'box-shadow: 0 4px 6px rgba(0,0,0,0.1);' +
                'z-index: 1000;' +
                'animation: slideIn 0.3s ease-out;';
            
            msgDiv.innerHTML = 
                '<div style="font-weight: bold; margin-bottom: 5px;">⚠️ 服务器已断开</div>' +
                '<div style="font-size: 14px;">监控服务已停止，请刷新页面重试</div>';
            
            // 添加动画样式
            const style = document.createElement('style');
            style.textContent = 
                '@keyframes slideIn {' +
                '    from {' +
                '        transform: translateX(400px);' +
                '        opacity: 0;' +
                '    }' +
                '    to {' +
                '        transform: translateX(0);' +
                '        opacity: 1;' +
                '    }' +
                '}';
            
            document.head.appendChild(style);
            container.appendChild(msgDiv);
        }

        function updateStats(data) {
            if (data.length === 0) return;
            
            const latest = data[data.length - 1];
            const avgPause = data.reduce((sum, d) => sum + d.pause_ms, 0) / data.length;
            const maxPause = Math.max(...data.map(d => d.pause_ms));
            
            if (!statsInitialized) {
                // 首次初始化，创建完整的 HTML 结构
                document.getElementById('stats').innerHTML = 
	'<div class="stat-card">' +
	'<div class="stat-value" id="stat-num-gc">' + latest.num_gc + '</div>' +
	'<div class="stat-label">GC 次数</div>' +
	'<div class="tooltip">累计执行的垃圾回收次数。GC 会自动清理不再使用的内存对象。</div>' +
	'</div>' +
	'<div class="stat-card">' +
	'<div class="stat-value" id="stat-pause-ms">' + latest.pause_ms.toFixed(2) + 'ms</div>' +
	'<div class="stat-label">最新暂停时间</div>' +
	'<div class="tooltip">最近一次 GC 导致的程序暂停时间。Go 的 GC 是并发标记清除，暂停时间通常很短。</div>' +
	'</div>' +
	'<div class="stat-card">' +
	'<div class="stat-value" id="stat-avg-pause">' + latest.avg_pause_ms.toFixed(2) + 'ms</div>' +
	'<div class="stat-label">平均暂停时间</div>' +
	'<div class="tooltip">所有 GC 暂停时间的平均值。比单次暂停更能反映整体 GC 性能。</div>' +
	'</div>' +
	'<div class="stat-card">' +
	'<div class="stat-value" id="stat-max-pause">' + maxPause.toFixed(2) + 'ms</div>' +
	'<div class="stat-label">最大暂停时间</div>' +
	'<div class="tooltip">监控期间记录到的最长 GC 暂停时间。用于评估 GC 的最坏情况。</div>' +
	'</div>' +
	'<div class="stat-card">' +
	'<div class="stat-value" id="stat-heap-alloc">' + latest.heap_alloc_mb.toFixed(2) + 'MB</div>' +
	'<div class="stat-label">堆分配</div>' +
	'<div class="tooltip">当前已分配的堆内存总量。包括正在使用和空闲的内存。</div>' +
	'</div>' +
	'<div class="stat-card">' +
	'<div class="stat-value" id="stat-utilization">' + latest.heap_utilization.toFixed(1) + '%</div>' +
	'<div class="stat-label">堆利用率</div>' +
	'<div class="tooltip">堆内存使用率 = 使用中内存 / 系统分配内存 × 100%。过低可能存在内存碎片。</div>' +
	'</div>' +
	'<div class="stat-card">' +
	'<div class="stat-value" id="stat-objects">' + latest.heap_objects.toLocaleString() + '</div>' +
	'<div class="stat-label">活跃对象</div>' +
	'<div class="tooltip">当前堆中的活跃对象数量。持续增长可能表示内存泄漏。</div>' +
	'</div>' +
	'<div class="stat-card">' +
	'<div class="stat-value" id="stat-cpu-percent">' + latest.gc_cpu_percent.toFixed(2) + '%</div>' +
	'<div class="stat-label">GC CPU 占用</div>' +
	'<div class="tooltip">GC 占用的 CPU 时间比例。过高会影响应用程序性能。</div>' +
	'</div>';
                statsInitialized = true;
            } else {
                // 后续只更新数值，不重建 DOM
                document.getElementById('stat-num-gc').textContent = latest.num_gc;
                document.getElementById('stat-pause-ms').textContent = latest.pause_ms.toFixed(2) + 'ms';
                document.getElementById('stat-avg-pause').textContent = latest.avg_pause_ms.toFixed(2) + 'ms';
                document.getElementById('stat-max-pause').textContent = maxPause.toFixed(2) + 'ms';
                document.getElementById('stat-heap-alloc').textContent = latest.heap_alloc_mb.toFixed(2) + 'MB';
                document.getElementById('stat-utilization').textContent = latest.heap_utilization.toFixed(1) + '%';
                document.getElementById('stat-objects').textContent = latest.heap_objects.toLocaleString();
                document.getElementById('stat-cpu-percent').textContent = latest.gc_cpu_percent.toFixed(2) + '%';
            }
        }

        function updateCharts(data) {
            const labels = data.map(d => new Date(d.time).toLocaleTimeString());
            
            if (!pauseChart) {
                createCharts(labels, data);
            } else {
                pauseChart.data.labels = labels;
                pauseChart.data.datasets[0].data = data.map(d => d.pause_ms);
                pauseChart.data.datasets[1].data = data.map(d => d.avg_pause_ms);
                pauseChart.update();
                
                heapChart.data.labels = labels;
                heapChart.data.datasets[0].data = data.map(d => d.heap_alloc_mb);
                heapChart.data.datasets[1].data = data.map(d => d.heap_inuse_mb);
                heapChart.update();
                
                cpuChart.data.labels = labels;
                cpuChart.data.datasets[0].data = data.map(d => d.gc_cpu_percent);
                cpuChart.update();
                
                utilizationChart.data.labels = labels;
                utilizationChart.data.datasets[0].data = data.map(d => d.heap_utilization);
                utilizationChart.update();
                
                objectsChart.data.labels = labels;
                objectsChart.data.datasets[0].data = data.map(d => d.heap_objects);
                objectsChart.update();
                
                releasedChart.data.labels = labels;
                releasedChart.data.datasets[0].data = data.map(d => d.heap_released_mb);
                releasedChart.update();
            }
        }

        function createCharts(labels, data) {
            const commonOptions = {
                responsive: true,
                maintainAspectRatio: true,
                plugins: {
                    legend: {
                        display: true,
                        position: 'top'
                    },
                    title: {
                        display: true,
                        font: {
                            size: 16
                        }
                    }
                },
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: '时间'
                        }
                    },
                    y: {
                        display: true,
                        beginAtZero: true
                    }
                }
            };

            pauseChart = new Chart(document.getElementById('pauseChart'), {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [
                        {
                            label: 'GC 暂停时间 (ms)',
                            data: data.map(d => d.pause_ms),
                            borderColor: 'rgb(255, 99, 132)',
                            backgroundColor: 'rgba(255, 99, 132, 0.1)',
                            tension: 0.1,
                            fill: true
                        },
                        {
                            label: '平均暂停时间 (ms)',
                            data: data.map(d => d.avg_pause_ms),
                            borderColor: 'rgb(255, 159, 64)',
                            backgroundColor: 'rgba(255, 159, 64, 0.1)',
                            tension: 0.1,
                            fill: false,
                            borderDash: [5, 5]
                        }
                    ]
                },
                options: {
                    ...commonOptions,
                    plugins: {
                        ...commonOptions.plugins,
                        title: {
                            display: true,
                            text: 'GC 暂停时间趋势'
                        }
                    }
                }
            });

            heapChart = new Chart(document.getElementById('heapChart'), {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [
                        {
                            label: '堆分配 (MB)',
                            data: data.map(d => d.heap_alloc_mb),
                            borderColor: 'rgb(54, 162, 235)',
                            backgroundColor: 'rgba(54, 162, 235, 0.1)',
                            tension: 0.1,
                            fill: true
                        },
                        {
                            label: '堆使用 (MB)',
                            data: data.map(d => d.heap_inuse_mb),
                            borderColor: 'rgb(75, 192, 192)',
                            backgroundColor: 'rgba(75, 192, 192, 0.1)',
                            tension: 0.1,
                            fill: true
                        }
                    ]
                },
                options: {
                    ...commonOptions,
                    plugins: {
                        ...commonOptions.plugins,
                        title: {
                            display: true,
                            text: '堆内存使用情况'
                        }
                    }
                }
            });

            releasedChart = new Chart(document.getElementById('releasedChart'), {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [{
                        label: '已释放内存 (MB)',
                        data: data.map(d => d.heap_released_mb),
                        borderColor: 'rgb(255, 99, 132)',
                        backgroundColor: 'rgba(255, 99, 132, 0.1)',
                        tension: 0.1,
                        fill: true
                    }]
                },
                options: {
                    ...commonOptions,
                    plugins: {
                        ...commonOptions.plugins,
                        title: {
                            display: true,
                            text: '内存释放趋势'
                        }
                    }
                }
            });

            cpuChart = new Chart(document.getElementById('cpuChart'), {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [{
                        label: 'GC CPU 占用 (%)',
                        data: data.map(d => d.gc_cpu_percent),
                        borderColor: 'rgb(255, 159, 64)',
                        backgroundColor: 'rgba(255, 159, 64, 0.1)',
                        tension: 0.1,
                        fill: true
                    }]
                },
                options: {
                    ...commonOptions,
                    plugins: {
                        ...commonOptions.plugins,
                        title: {
                            display: true,
                            text: 'GC CPU 占用率'
                        }
                    }
                }
            });

            utilizationChart = new Chart(document.getElementById('utilizationChart'), {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [{
                        label: '堆利用率 (%)',
                        data: data.map(d => d.heap_utilization),
                        borderColor: 'rgb(75, 192, 192)',
                        backgroundColor: 'rgba(75, 192, 192, 0.1)',
                        tension: 0.1,
                        fill: true
                    }]
                },
                options: {
                    ...commonOptions,
                    plugins: {
                        ...commonOptions.plugins,
                        title: {
                            display: true,
                            text: '堆内存利用率'
                        }
                    },
                    scales: {
                        ...commonOptions.scales,
                        y: {
                            display: true,
                            beginAtZero: true,
                            max: 100,
                            title: {
                                display: true,
                                text: '利用率 (%)'
                            }
                        }
                    }
                }
            });

            objectsChart = new Chart(document.getElementById('objectsChart'), {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [{
                        label: '活跃对象数量',
                        data: data.map(d => d.heap_objects),
                        borderColor: 'rgb(153, 102, 255)',
                        backgroundColor: 'rgba(153, 102, 255, 0.1)',
                        tension: 0.1,
                        fill: true
                    }]
                },
                options: {
                    ...commonOptions,
                    plugins: {
                        ...commonOptions.plugins,
                        title: {
                            display: true,
                            text: '活跃对象数量趋势'
                        }
                    }
                }
            });
        }

        fetchData();
        pollingInterval = setInterval(fetchData, 2000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func serveData(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(dataPoints)
	if err != nil {
		return
	}
}
