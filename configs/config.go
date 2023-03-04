package configs

import (
	"encoding/json"
	"flag"
	"github.com/BurntSushi/toml"
	"github.com/lesismal/nbio/logging"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	//日志级别 debug、info、warn、error
	LogLevel string
	//服务唯一编号，取值范围是 uint8，会成为uniqId的前缀
	//如果服务编号不是唯一的，则多个网关机器可能生成相同的uniqId，但是，如果业务层不关心这个uniqId，比如根据uniqId前缀判断在哪个网关机器，则不必考虑服务编号唯一性
	ServerId uint8

	//business的限流设置，min、max的取值范围是1~999,表示的就是business的workerId
	Limit []Limit

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
	}
	Worker struct {
		//worker服务器监听的地址，ip:port，这个地址最好是内网地址，外网不允许访问
		ListenAddress string
		//worker读取business连接的超时时间，该时间段内，business连接没有发消息过来，则会超时，连接会被关闭
		ReadDeadline time.Duration
		//worker读取business的包的大小限制（单位：字节）
		ReceivePackLimit uint32
		//worker发送给business连接的超时时间，该时间段内，没发送成功，business连接会被关闭
		SendDeadline time.Duration
		//worker发送给business的包的大小限制（单位：字节）
		SendPackLimit uint32
	}

	Metrics struct {
		//统计服务的各种状态，空，则不统计任何状态，0：统计客户连接的打开情况，1：统计客户连接的关闭情况，2：统计客户连接的心跳情况，3：统计客户数据转发到worker的情况
		Item []int
		//统计服务的各种状态里记录最大值的间隔时间（单位：秒）
		MaxRecordInterval time.Duration
	}
}

type Limit struct {
	Min int
	Max int
	Num int
}

func (r Limit) String() string {
	b, _ := json.Marshal(r)
	return string(b)
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

// RootPath 程序根目录
var RootPath string

func init() {
	dir, err := os.Getwd()
	if err != nil {
		logging.Error("获取进程工作目录失败：%s", err)
		os.Exit(1)
	}
	RootPath = strings.TrimSuffix(filepath.ToSlash(dir), "/") + "/"
}

// Config 应用程序配置
var Config *config

func init() {
	var configFile string
	flag.StringVar(&configFile, "config", filepath.Join(RootPath, "configs/config.toml"), "set configuration file")
	flag.Parse()
	//读取配置文件
	c, err := os.ReadFile(configFile)
	if err != nil {
		logging.Error("读取配置失败：%s", err)
		os.Exit(1)
	}
	//解析配置文件到对象
	Config = new(config)
	if _, err := toml.Decode(string(c), Config); err != nil {
		logging.Error("配置解析错误：%s", err)
		os.Exit(1)
	}
	//检查各种参数
	if Config.Customer.MaxOnlineNum <= 0 {
		//默认最多负载十万个连接，超过的会被拒绝
		Config.Customer.MaxOnlineNum = 10 * 10000
	}
	if Config.Customer.ReadDeadline <= 0 {
		//默认120秒
		Config.Customer.ReadDeadline = time.Second * 120
	}
	if Config.Worker.ReadDeadline <= 0 {
		//默认120秒
		Config.Worker.ReadDeadline = time.Second * 120
	}
	if Config.Worker.SendDeadline <= 0 {
		//默认10秒
		Config.Worker.SendDeadline = time.Second * 10
	}
	if Config.Worker.ReceivePackLimit <= 0 {
		//默认2MB
		Config.Worker.ReceivePackLimit = 2 * 1024 * 1024
	}
	if Config.Worker.SendPackLimit <= 0 {
		//默认2MB
		Config.Worker.SendPackLimit = 2 * 1024 * 1024
	}
	if Config.Metrics.MaxRecordInterval <= 0 {
		//默认10秒
		Config.Metrics.MaxRecordInterval = time.Second * 10
	}
}
