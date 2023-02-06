package configs

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/lesismal/nbio/logging"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	//客户服务器监听的地址，ip:port，这个地址一般是外网地址
	CustomerListenAddress string
	//客户服务器的url路由
	CustomerHandlePattern string
	//网关检查客户连接的心跳的时间间隔（单位：秒）
	CustomerHeartbeatIntervalSecond int64

	//session id的最小值，包含该值，不能为0
	SessionIdMin uint32
	//session id的最大值，包含该值
	SessionIdMax uint32

	//worker服务器监听的地址，ip:port，这个地址最好是内网地址，外网不允许访问
	WorkerListenAddress string
	//worker检查business连接的心跳的时间间隔（单位：秒）
	WorkerHeartbeatIntervalSecond int64
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
	MetricsMaxRecordIntervalSecond int
}

// rootPath 程序根目录
var rootPath string

func init() {
	var dir string
	var err error
	//兼容GoLand编辑器下的go run命令
	tmp := strings.ToLower(os.Args[0])
	fmt.Println(tmp)
	features := []string{"go_build", "go-build", "tmp", "temp"}
	isGoBuildRun := false
	for _, v := range features {
		if strings.Contains(tmp, v) {
			isGoBuildRun = true
			break
		}
	}
	if isGoBuildRun {
		dir, err = os.Getwd()
		//兼容go run .\cmd\main.go
		if err == nil && strings.HasSuffix(dir, "cmd") {
			dir = filepath.Dir(dir)
		}
	} else {
		dir, err = filepath.Abs(filepath.Dir(os.Args[0]))
	}
	if err != nil {
		logging.Error("获取进程工作目录失败：%s", err)
		os.Exit(1)
	}
	rootPath = strings.TrimSuffix(filepath.ToSlash(dir), "/") + "/"
}

// Config 应用程序配置
var Config *config

func init() {
	var configFile string
	flag.StringVar(&configFile, "config", filepath.Join(rootPath, "configs/config.toml"), "set configuration file")
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
	if Config.SessionIdMin < 1 {
		logging.Error("session id的最小值不能为0")
		os.Exit(1)
	}
	if Config.SessionIdMax < Config.SessionIdMin {
		logging.Error("session id的最大值配置，必须大于session id的最小值配置")
		os.Exit(1)
	}
	if Config.WorkerHeartbeatIntervalSecond <= 0 {
		//默认55秒
		Config.WorkerHeartbeatIntervalSecond = 55
	}
	if Config.WorkerReceivePackLimit <= 0 {
		//默认2MB
		Config.WorkerReceivePackLimit = 2 * 1024 * 1024
	}
	if Config.WorkerSendPackLimit <= 0 {
		//默认2MB
		Config.WorkerSendPackLimit = 2 * 1024 * 1024
	}
	if Config.CustomerHeartbeatIntervalSecond <= 0 {
		//默认55秒
		Config.CustomerHeartbeatIntervalSecond = 55
	}
	if Config.CatapultConsumer <= 0 {
		//默认1000个协程
		Config.CatapultConsumer = 1000
	}
	if Config.CatapultChanCap <= 0 {
		//缓冲区默认大小是2000
		Config.CatapultChanCap = 2000
	}
	if Config.MetricsMaxRecordIntervalSecond <= 0 {
		//默认10秒
		Config.MetricsMaxRecordIntervalSecond = 10
	}
}
