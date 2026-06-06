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
	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog"
	"log/slog"
	"netsvr/pkg/wd"
	"os"
	"path/filepath"
)

type BytesConfigItem []byte

func (r *BytesConfigItem) UnmarshalText(text []byte) error {
	*r = text
	return nil
}

type RedisQueue struct {
	//Redis地址
	Address string
	//Redis密码
	Password string
	//Redis数据库
	DB int
	//Redis队列的key
	Key string
	//Redis队列的key类型，目前支持：stream、list
	KeyType string
}

// AMQP091Queue AMQP队列的配置
type AMQP091Queue struct {
	//RabbitMQ服务器地址 host:port
	Address string
	//用户名
	Username string
	//密码
	Password string
	//虚拟主机（默认 /）
	VHost string
	//Exchange名称
	Exchange string
	//Exchange类型: direct, fanout, topic, headers
	ExchangeType string
	//Queue名称（空则不声明Queue，仅发布到Exchange）
	Queue string
	//Routing Key
	RoutingKey string
	//是否持久化
	Durable bool
	//是否自动删除
	AutoDelete bool
}

type config struct {
	//日志级别 debug、info、warn、error
	LogLevel string
	//提供服务的方式，目前支持：worker、queue、callback
	Service string
	//worker服务的监听地址
	WorkerListenAddress string
	//任务服务的监听地址
	TaskListenAddress string
	//customer服务的websocket连接地址
	ClientListenAddress string
	//输出客户端界面的http服务的监听地址
	CustomerWsAddress string
	//客户端websocket发送的心跳消息
	CustomerHeartbeatMessage BytesConfigItem
	//business进程向网关的worker服务器发送的心跳消息
	WorkerHeartbeatMessage BytesConfigItem
	//business进程向网关的task服务器发送的心跳消息
	TaskHeartbeatMessage BytesConfigItem
	//Redis队列的配置
	RedisQueue struct {
		//连接打开的Redis队列
		OnOpen RedisQueue
		//发送消息的Redis队列
		OnMessage RedisQueue
		//连接关闭的Redis队列
		OnClose RedisQueue
	}
	//AMQP队列的配置
	AMQP091Queue AMQP091Queue
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

var Config *config

func init() {
	var configFile string
	flag.StringVar(&configFile, "config", filepath.Join(wd.RootPath, "configs/business.toml"), "Set business.toml file")
	flag.Parse()
	//读取配置文件
	c, err := os.ReadFile(configFile)
	if err != nil {
		slog.Error("Read business.toml failed", "error", err)
		os.Exit(1)
	}
	//解析配置文件到对象
	Config = new(config)
	if _, err := toml.Decode(string(c), Config); err != nil {
		slog.Error("Parse business.toml failed", "error", err)
		os.Exit(1)
	}
	if Config.Service == "" {
		Config.Service = "worker"
	}
	if Config.Service != "worker" && Config.Service != "redis" && Config.Service != "amqp091" && Config.Service != "callback" {
		slog.Error("Invalid service", "service", Config.Service)
		os.Exit(1)
	}
}
