package configs

import (
	"flag"
	"github.com/BurntSushi/toml"
	"github.com/lesismal/nbio/logging"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	//服务唯一编号，取值范围是 uint8，会成为uniqId的前缀
	//如果服务编号不是唯一的，则多个网关机器可能生成相同的uniqId，但是，如果业务层不关心这个uniqId，比如根据uniqId前缀判断在哪个网关机器，则不必考虑服务编号唯一性
	ServerId uint8
	//客户服务器监听的地址，ip:port，这个地址一般是外网地址
	CustomerListenAddress string
	//客户服务器的url路由
	CustomerHandlePattern string
	//网关读取客户连接的超时时间，该时间段内，客户连接没有发消息过来，则会超时，连接会被关闭
	CustomerReadDeadline time.Duration
	//最大连接数
	CustomerMaxOnlineNum int

	//worker服务器监听的地址，ip:port，这个地址最好是内网地址，外网不允许访问
	WorkerListenAddress string
	//worker读取business连接的超时时间，该时间段内，business连接没有发消息过来，则会超时，连接会被关闭
	WorkerReadDeadline time.Duration
	//worker读取business的包的大小限制（单位：字节）
	WorkerReceivePackLimit uint32
	//worker发送给business的包的大小限制（单位：字节）
	WorkerSendPackLimit uint32

	//网关用于发送数据给客户的协程数量
	CatapultConsumer int
	//等待发送给客户数据的缓冲区的大小
	CatapultChanCap int

	//统计服务的各种状态，空，则不统计任何状态，0：统计客户连接的打开情况，1：统计客户连接的关闭情况，2：统计客户连接的心跳情况，3：统计客户数据转发到worker的情况
	MetricsItem []int
	//统计服务的各种状态里记录最大值的间隔时间（单位：秒）
	MetricsMaxRecordInterval time.Duration
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
	if Config.WorkerReadDeadline <= 0 {
		//默认120秒
		Config.WorkerReadDeadline = time.Second * 120
	}
	if Config.CustomerMaxOnlineNum <= 0 {
		//默认最多负载十万个连接
		Config.CustomerMaxOnlineNum = 10 * 10000
	}
	if Config.WorkerReceivePackLimit <= 0 {
		//默认2MB
		Config.WorkerReceivePackLimit = 2 * 1024 * 1024
	}
	if Config.WorkerSendPackLimit <= 0 {
		//默认2MB
		Config.WorkerSendPackLimit = 2 * 1024 * 1024
	}
	if Config.CustomerReadDeadline <= 0 {
		//默认120秒
		Config.CustomerReadDeadline = time.Second * 120
	}
	if Config.CatapultConsumer <= 0 {
		//默认1000个协程
		Config.CatapultConsumer = 1000
	}
	if Config.CatapultChanCap <= 0 {
		//缓冲区默认大小是2000
		Config.CatapultChanCap = 2000
	}
	if Config.MetricsMaxRecordInterval <= 0 {
		//默认10秒
		Config.MetricsMaxRecordInterval = time.Second * 10
	}
}
