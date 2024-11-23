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

package configs

import (
	"compress/flate"
	_ "embed"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"github.com/rs/zerolog"
	"log/slog"
	"net"
	"netsvr/internal/utils"
	"netsvr/pkg/wd"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type BytesConfigItem []byte

func (r *BytesConfigItem) UnmarshalText(text []byte) error {
	*r = text
	return nil
}

type config struct {
	//日志级别 debug、info、warn、error
	LogLevel string
	// 日志文件，如果不配置，则输出到控制台，如果配置的是相对地址（不包含盘符或不是斜杠开头），则会自动拼接上进程工作目录的地址，最好配置绝对路径
	LogFile string
	//网关收到停止信号后的等待时间，0表示永久等待，否则是超过这个时间还没优雅停止，则会强制退出
	ShutdownWaitTime time.Duration
	//pprof服务器监听的地址，ip:port，这个地址必须是内网地址，外网不允许访问，如果是空字符串，则不会开启，生产环境服务没毛病就别开它
	PprofListenAddress string
	//限流器，如果配置是0，则表示不启用限流器
	Limit struct {
		//网关允许每秒打开多少个连接，配置为0，则不做任何限制
		OnOpen int32
		//网关允许每秒收到多少个消息，配置为0，则不做任何限制
		OnMessage int32
	}
	//客户端的websocket服务器配置
	Customer struct {
		//客户服务器监听的地址，ip:port，这个地址一般是外网地址
		ListenAddress string
		//客户服务器的url路由
		HandlePattern string
		//允许连接的origin，空表示不限制，否则会进行包含匹配
		AllowOrigin []string
		//websocket服务器读取客户端连接的超时时间，该时间段内，客户端连接没有发消息过来，则会超时，连接会被关闭，所以客户端连接必须在该时间间隔内发送心跳字符串
		ReadDeadline time.Duration
		//最大连接数，超过的会被拒绝
		MaxOnlineNum int
		//io模式，0：IOModNonBlocking，1：IOModBlocking，2：IOModMixed，详细见：https://github.com/lesismal/nbio
		IOMod int
		//最大阻塞连接数，IOMod是2时有效，详细见：https://github.com/lesismal/nbio
		MaxBlockingOnline int
		//客户发送数据的大小限制（单位：字节）
		ReceivePackLimit int
		//往websocket连接写入时的消息类型，1：TextMessage，2：BinaryMessage
		SendMessageType websocket.MessageType
		// 回调脚本文件，如果需要，最好配置绝对路径，不需要，请配置为空字符串
		CallbackScriptFile string
		//tls配置
		TLSCert string
		TLSKey  string
		//心跳字符串，客户端连接必须定时发送该字符串，用于维持心跳
		HeartbeatMessage BytesConfigItem
		//压缩级别，区间是：[-2,9]，0表示不压缩，具体见https://golang.org/pkg/compress/flate/
		CompressionLevel int
	}
	//业务进程的tcp服务器配置
	Worker struct {
		// 监听的地址，ipv4:port，这个地址必须是内网ipv4地址，外网不允许访问，如果配置的是域名:端口，则会尝试获取域名对应的内网ipv4地址，并打印告警日志
		ListenAddress string
		//worker读取business连接的超时时间，该时间段内，business连接没有发消息过来，则会超时，连接会被关闭
		ReadDeadline time.Duration
		//worker发送给business连接的超时时间，该时间段内，没发送成功，business连接会被关闭
		SendDeadline time.Duration
		//business发送数据的大小限制（单位：字节），business发送了超过该限制的包，则连接会被关闭
		ReceivePackLimit uint32
		//读取business发送数据的缓冲区大小（单位：字节）
		ReadBufferSize int
		//worker发送给business的缓通道大小
		SendChanCap int
		//心跳字符串，客户端连接必须定时发送该字符串，用于维持心跳
		HeartbeatMessage BytesConfigItem
	}

	Metrics struct {
		// 统计服务的各种状态，空，则不统计任何状态
		// 1：统计客户连接的打开次数
		// 2：统计客户连接的关闭次数
		// 3：统计客户连接的心跳次数
		// 4：统计客户数据转发到worker的次数
		// 5：统计客户数据转发到worker的字节数
		// 6：统计往客户写入数据次数
		// 7：统计往客户写入字节数
		// 8：统计连接打开的限流次数
		// 9：统计客户消息限流次数
		// 10：统计worker到business的失败次数
		Item []int
	}
}

func (r *config) GetLogFile() string {
	if r.LogFile == "" {
		return ""
	}
	if strings.HasPrefix(r.LogFile, "/") || r.LogFile[1:2] == ":" {
		//绝对路径开头
		return r.LogFile
	}
	//拼接上相对路径
	return filepath.Join(wd.RootPath, r.LogFile)
}

func (r *config) GetLogLevel() zerolog.Level {
	switch r.LogLevel {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	}
	return zerolog.ErrorLevel
}

// Config 应用程序配置
var Config *config

// 构建版本信息
var (
	BuildVersion   = ""
	BuildGoVersion = ""
	BuildTimestamp = ""
)

func init() {
	var configFile string
	var version bool
	flag.BoolVar(&version, "version", false, "Version prints the build information for netsvr binary files.")
	flag.StringVar(&configFile, "config", filepath.Join(wd.RootPath, "configs/netsvr.toml"), "Set netsvr.toml file")
	flag.Parse()
	//打印构建信息
	if version {
		t, err := strconv.ParseInt(BuildTimestamp, 10, 0)
		if err != nil {
			slog.Error("Prints the build information failed", "error", err)
			os.Exit(1)
		}
		fmt.Printf("Go version: %s\nBuild version: %s\nBuild date: %s\n", BuildGoVersion, BuildVersion, time.Unix(t, 0).Format(time.RFC3339))
		os.Exit(0)
	}
	//读取配置文件
	c, err := os.ReadFile(configFile)
	if err != nil {
		slog.Error("Read netsvr.toml failed", "error", err)
		os.Exit(1)
	}
	//解析配置文件到对象
	Config = new(config)
	if _, err := toml.Decode(string(c), Config); err != nil {
		slog.Error("Parse netsvr.toml failed", "error", err)
		os.Exit(1)
	}
	//检查各种参数
	if Config.Customer.MaxOnlineNum <= 0 {
		//默认最多负载十万个连接，超过的会被拒绝
		Config.Customer.MaxOnlineNum = 10 * 10000
	}
	if Config.Customer.IOMod != nbhttp.IOModNonBlocking && Config.Customer.IOMod != nbhttp.IOModBlocking && Config.Customer.IOMod != nbhttp.IOModMixed {
		Config.Customer.IOMod = nbhttp.DefaultIOMod
	}
	if Config.Customer.MaxBlockingOnline <= 0 {
		Config.Customer.MaxBlockingOnline = nbhttp.DefaultMaxBlockingOnline
	}
	//默认只接收2MiB以内的数据
	if Config.Customer.ReceivePackLimit <= 0 {
		Config.Customer.ReceivePackLimit = 2097152
	}
	//检查发消息的类型
	if Config.Customer.SendMessageType == 0 {
		Config.Customer.SendMessageType = websocket.TextMessage
	} else if Config.Customer.SendMessageType != websocket.TextMessage && Config.Customer.SendMessageType != websocket.BinaryMessage {
		slog.Error(fmt.Sprintf("Config Customer.SendMessageType, must be in [%d,%d]", websocket.TextMessage, websocket.BinaryMessage))
		os.Exit(1)
	}
	if Config.Customer.ReadDeadline <= 0 {
		//默认120秒
		Config.Customer.ReadDeadline = time.Second * 120
	}
	if Config.Customer.HandlePattern == "" {
		Config.Customer.HandlePattern = "/"
	}
	if Config.Customer.CallbackScriptFile != "" {
		if ok, err := utils.IsFile(Config.Customer.CallbackScriptFile); !ok {
			slog.Error("Config Customer.CallbackScriptFile is not a file", "error", err)
			os.Exit(1)
		}
	}
	if len(Config.Customer.HeartbeatMessage) == 0 {
		slog.Error("Config Customer.HeartbeatMessage is required")
		os.Exit(1)
	}
	if Config.Customer.CompressionLevel < flate.HuffmanOnly || Config.Customer.CompressionLevel > flate.BestCompression {
		slog.Error("Config Customer.CompressionLevel is invalid")
		os.Exit(1)
	}
	if Config.Worker.ReadDeadline <= 0 {
		//默认120秒
		Config.Worker.ReadDeadline = time.Second * 120
	}
	if Config.Worker.SendDeadline <= 0 {
		//默认10秒
		Config.Worker.SendDeadline = time.Second * 10
	}
	if Config.Worker.ReceivePackLimit == 0 {
		Config.Worker.ReceivePackLimit = 1024 * 1024 * 2
	}
	if Config.Worker.ReadBufferSize <= 0 {
		Config.Worker.ReadBufferSize = 4 * 1024
	}
	if Config.Worker.SendChanCap <= 0 {
		Config.Worker.SendChanCap = 1024
	}
	//检查网关的worker服务的监听地址
	if host, port, err := net.SplitHostPort(Config.Worker.ListenAddress); err != nil {
		slog.Error("Config Worker.ListenAddress, split failed", "error", err)
		os.Exit(1)
	} else {
		if !utils.IsValidIPv4(host) {
			//将域名转为内网地址
			slog.Warn("Config Worker.ListenAddress, host is not a valid ipv4 address", "host", host)
			ipv4 := utils.GetHostByName(host)
			if ipv4 == host {
				slog.Warn("Config Worker.ListenAddress, convert " + host + " to ipv4 failed")
				os.Exit(1)
			}
			slog.Warn("Config Worker.ListenAddress, convert " + host + " to " + ipv4 + " successful")
			Config.Worker.ListenAddress = net.JoinHostPort(ipv4, port)
		} else if host == "0.0.0.0" {
			//转为内网地址
			ipv4 := utils.GetLocalIPAddress()
			if ipv4 == "" {
				slog.Error("Config Worker.ListenAddress, Not allowed to configure 0.0.0.0")
				os.Exit(1)
			}
			slog.Warn("Config Worker.ListenAddress, convert " + host + " to " + ipv4 + " successful")
			Config.Worker.ListenAddress = net.JoinHostPort(ipv4, port)
		}
	}
	if len(Config.Worker.HeartbeatMessage) == 0 {
		slog.Error("Config Worker.HeartbeatMessage is required")
		os.Exit(1)
	}
	fileIsExist := func(path string) bool {
		fi, err := os.Stat(path)
		if err == nil {
			return !fi.IsDir()
		}
		if os.IsNotExist(err) {
			return false
		}
		return true
	}
	if Config.Customer.TLSKey != "" && !fileIsExist(Config.Customer.TLSKey) {
		Config.Customer.TLSKey = filepath.Join(wd.RootPath, "configs/"+Config.Customer.TLSKey)
	}
	if Config.Customer.TLSCert != "" && !fileIsExist(Config.Customer.TLSCert) {
		Config.Customer.TLSCert = filepath.Join(wd.RootPath, "configs/"+Config.Customer.TLSCert)
	}
}
