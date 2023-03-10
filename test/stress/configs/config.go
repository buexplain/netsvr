package configs

import (
	"flag"
)

var Config *config

func init() {
	var configFile string
	flag.StringVar(&configFile, "config", "stress.toml", "Set stress.toml file")
	flag.Parse()
}

type config struct {
	//沉默的大多数
	Silent []Step
	//疯狂的登录登出
	Sign []Step
	//疯狂单发
	SingleCast []Step
	//疯狂群发
	MulticastCast []Step
	//疯狂广播
	Broadcast []Step
	//疯狂订阅、取消订阅、发布
	Topic []Step
}

type Step struct {
	//连接数
	ConnNum int
	//发起连接的间隔，单位毫秒
	ConnectInterval int
	//连接完成后，暂停时间，该时间后进入下一个step
	Suspend int
	//发送的消息大小
	MessageLen int
	//发消息的间隔，单位毫秒
	MessageInterval int
	//可以执行的方法名单
	Method []string
}
