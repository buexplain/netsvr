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
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"github.com/rs/zerolog"
	"netsvr/pkg/wd"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type config struct {
	//日志级别 debug、info、warn、error
	LogLevel string
	// 日志文件，如果不配置，则输出到控制台，如果配置的是相对地址（不包含盘符或不是斜杠开头），则会自动拼接上进程工作目录的地址
	LogFile string
	//网关服务唯一编号，取值范围是 uint8，会成为uniqId的前缀，16进制表示，占两个字符
	ServerId uint8
	//网关收到停止信号后的等待时间，0表示永久等待，否则是超过这个时间还没优雅停止，则会强制退出
	ShutdownWaitTime time.Duration
	//pprof服务器监听的地址，ip:port，这个地址必须是内网地址，外网不允许访问，如果是空值，则不会开启
	PprofListenAddress string
	//business的限流器
	Limit []struct {
		//需要被限制的workerId集合，每个workerId只允许配置一次
		WorkerIds []int
		//每秒的并发数
		Concurrency int
		//限流器名字，这个名字必须是唯一的
		Name string
	}
	Customer struct {
		//客户服务器监听的地址，ip:port，这个地址一般是外网地址
		ListenAddress string
		//客户服务器的url路由
		HandlePattern string
		//允许连接的origin，空表示不限制，否则会进行包含匹配
		AllowOrigin []string
		//网关读取客户连接的超时时间，该时间段内，客户连接没有发消息过来，则会超时，连接会被关闭
		ReadDeadline time.Duration
		//最大连接数，超过的会被拒绝
		MaxOnlineNum int
		//客户发送数据的大小限制（单位：字节）
		ReceivePackLimit int
		//往websocket连接写入时的消息类型，1：TextMessage，2：BinaryMessage
		SendMessageType websocket.MessageType
		//指定处理连接打开的worker id，允许设置为0，表示不关心连接的打开
		ConnOpenWorkerId int
		//指定处理连接关闭的worker id，允许设置为0，表示不关心连接的关闭
		ConnCloseWorkerId int
		//连接打开时，通过url的queryString传递自定义uniqId的字段名字，客户端如果传递该字段，则token必须一并传递过来
		//如果配置了该值，则网关会启用自定义uniqId的模式来接入连接
		//假设该值配置为 ConnOpenCustomUniqIdKey="myUniqId"，则客户端发起连接的示例：ws://127.0.0.1:6060/netsvr?myUniqId=01xxx&token=o0o0o
		ConnOpenCustomUniqIdKey string
		//自定义uniqId连接的token的过期时间
		ConnOpenCustomUniqIdTokenExpire time.Duration
		//tls配置
		TLSCert string
		TLSKey  string
	}

	Worker struct {
		//监听的地址，ip:port，这个地址必须是内网地址，外网不允许访问
		ListenAddress string
		//worker读取business连接的超时时间，该时间段内，business连接没有发消息过来，则会超时，连接会被关闭
		ReadDeadline time.Duration
		//worker发送给business连接的超时时间，该时间段内，没发送成功，business连接会被关闭
		SendDeadline time.Duration
		//business发送数据的大小限制（单位：字节）
		ReceivePackLimit uint32
	}

	Metrics struct {
		//统计服务的各种状态，空，则不统计任何状态，0：统计客户连接的打开情况，1：统计客户连接的关闭情况，2：统计客户连接的心跳情况，3：统计客户数据转发到worker的情况
		Item []int
		//统计服务的各种状态里记录最大值的间隔时间
		MaxRecordInterval time.Duration
	}
}

func (r *config) GetLogFile() string {
	if r.LogFile == "" {
		return ""
	}
	if strings.HasPrefix(r.LogFile, "/") {
		//绝对路径开头
		return r.LogFile
	}
	if r.LogFile[1:2] == ":" {
		//盘符开头
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
			logging.Error("Prints the build information failed：%s", err)
			os.Exit(1)
		}
		fmt.Printf("Go version: %s\nBuild version: %s\nBuild date: %s\n", BuildGoVersion, BuildVersion, time.Unix(t, 0).Format(time.RFC3339))
		os.Exit(0)
	}
	//读取配置文件
	c, err := os.ReadFile(configFile)
	if err != nil {
		logging.Error("Read netsvr.toml failed：%s", err)
		os.Exit(1)
	}
	//解析配置文件到对象
	Config = new(config)
	if _, err := toml.Decode(string(c), Config); err != nil {
		logging.Error("Parse netsvr.toml failed：%s", err)
		os.Exit(1)
	}
	//检查各种参数
	if Config.Customer.MaxOnlineNum <= 0 {
		//默认最多负载十万个连接，超过的会被拒绝
		Config.Customer.MaxOnlineNum = 10 * 10000
	}
	//默认只接收2MiB以内的数据
	if Config.Customer.ReceivePackLimit <= 0 {
		Config.Customer.ReceivePackLimit = 2097152
	}
	//检查发消息的类型
	if Config.Customer.SendMessageType == 0 {
		Config.Customer.SendMessageType = websocket.TextMessage
	} else if Config.Customer.SendMessageType != websocket.TextMessage && Config.Customer.SendMessageType != websocket.BinaryMessage {
		logging.Error("Config Customer.SendMessageType, must be in [%d,%d]", websocket.TextMessage, websocket.BinaryMessage)
		os.Exit(1)
	}
	if Config.Customer.ReadDeadline <= 0 {
		//默认120秒
		Config.Customer.ReadDeadline = time.Second * 120
	}
	if Config.Customer.HandlePattern == "" {
		Config.Customer.HandlePattern = "/"
	}
	if Config.Worker.ReadDeadline <= 0 {
		//默认120秒
		Config.Worker.ReadDeadline = time.Second * 120
	}
	if Config.Worker.SendDeadline <= 0 {
		//默认10秒
		Config.Worker.SendDeadline = time.Second * 10
	}
	if Config.Metrics.MaxRecordInterval <= 0 {
		//默认10秒
		Config.Metrics.MaxRecordInterval = time.Second * 10
	}
	//允许设置为0，表示不关心连接的打开
	if Config.Customer.ConnOpenWorkerId < netsvrProtocol.WorkerIdMin-1 || Config.Customer.ConnOpenWorkerId > netsvrProtocol.WorkerIdMax {
		logging.Error("Config Customer.ConnOpenWorkerId range overflow, must be in [%d,%d]", netsvrProtocol.WorkerIdMin-1, netsvrProtocol.WorkerIdMax)
		os.Exit(1)
	}
	//允许设置为0，表示不关心连接的关闭
	if Config.Customer.ConnCloseWorkerId < netsvrProtocol.WorkerIdMin-1 || Config.Customer.ConnCloseWorkerId > netsvrProtocol.WorkerIdMax {
		logging.Error("Config Customer.ConnCloseWorkerId range overflow, must be in [%d,%d]", netsvrProtocol.WorkerIdMin-1, netsvrProtocol.WorkerIdMax)
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
