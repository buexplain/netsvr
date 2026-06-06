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
	"strings"
)

type BytesConfigItem []byte

func (r *BytesConfigItem) UnmarshalText(text []byte) error {
	*r = text
	return nil
}

type RedisQueue struct {
	//Redis地址 host:port
	Address string
	//Redis密码
	Password *string
	//Redis数据库
	DB *int
	//Redis队列的key
	Key string
	//Redis队列的key类型，目前支持：stream、list(左侧压入消息)
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
	VHost *string
	//Exchange名称
	Exchange string
	//Exchange类型: direct, fanout, topic, headers
	ExchangeType string
	//Queue名称（空则不声明Queue，仅发布到Exchange）
	Queue *string
	//Routing Key
	RoutingKey *string
	//是否持久化
	Durable *bool
	//是否自动删除
	AutoDelete *bool
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
		RedisQueue
		//连接打开的Redis队列
		OnOpen RedisQueue
		//发送消息的Redis队列
		OnMessage RedisQueue
		//连接关闭的Redis队列
		OnClose RedisQueue
	}
	//AMQP091Queue队列的配置
	AMQP091Queue struct {
		//公共配置，会合并到 OnOpen、OnMessage、OnClose 配置节点中
		AMQP091Queue
		//连接打开的AMQP队列
		OnOpen AMQP091Queue
		//发送消息的AMQP队列
		OnMessage AMQP091Queue
		//连接关闭的AMQP队列
		OnClose AMQP091Queue
	}
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

	//设置redis队列的默认参数
	setRedisQueueDefaultParams := func(queue *RedisQueue) {
		//配置文件没有配置，则使用默认参数
		if queue.Address == "" {
			queue.Address = Config.RedisQueue.Address
		}
		if queue.Password == nil {
			queue.Password = Config.RedisQueue.Password
			if queue.Password == nil {
				queue.Password = new(string)
			}
		}
		if queue.DB == nil {
			queue.DB = Config.RedisQueue.DB
			if queue.DB == nil {
				queue.DB = new(int)
			}
		}
		if queue.Key == "" {
			queue.Key = Config.RedisQueue.Key
		}
		if queue.KeyType == "" {
			queue.KeyType = Config.RedisQueue.KeyType
		}
	}
	setRedisQueueDefaultParams(&Config.RedisQueue.OnOpen)
	setRedisQueueDefaultParams(&Config.RedisQueue.OnMessage)
	setRedisQueueDefaultParams(&Config.RedisQueue.OnClose)
	if Config.RedisQueue.OnOpen.KeyType == "" {
		Config.RedisQueue.OnOpen.KeyType = "list"
	} else {
		Config.RedisQueue.OnOpen.KeyType = strings.ToLower(Config.RedisQueue.OnOpen.KeyType)
		switch Config.RedisQueue.OnOpen.KeyType {
		case "list":
		case "stream":
		default:
			slog.Error("Config RedisQueue.OnOpen.KeyType is invalid")
			os.Exit(1)
		}
	}
	if Config.RedisQueue.OnClose.KeyType == "" {
		Config.RedisQueue.OnClose.KeyType = "list"
	} else {
		Config.RedisQueue.OnClose.KeyType = strings.ToLower(Config.RedisQueue.OnClose.KeyType)
		switch Config.RedisQueue.OnClose.KeyType {
		case "list":
		case "stream":
		default:
			slog.Error("Config RedisQueue.OnClose.KeyType is invalid")
			os.Exit(1)
		}
	}
	if Config.RedisQueue.OnMessage.KeyType == "" {
		Config.RedisQueue.OnMessage.KeyType = "list"
	} else {
		Config.RedisQueue.OnMessage.KeyType = strings.ToLower(Config.RedisQueue.OnMessage.KeyType)
		switch Config.RedisQueue.OnMessage.KeyType {
		case "list":
		case "stream":
		default:
			slog.Error("Config RedisQueue.OnMessage.KeyType is invalid")
		}
	}

	//设置amqp091队列的默认参数
	setAMQP091QueueDefaultParams := func(queue *AMQP091Queue) {
		if queue.Address == "" {
			queue.Address = Config.AMQP091Queue.Address
		}
		if queue.Username == "" {
			queue.Username = Config.AMQP091Queue.Username
		}
		if queue.Password == "" {
			queue.Password = Config.AMQP091Queue.Password
		}
		if queue.VHost == nil {
			queue.VHost = Config.AMQP091Queue.VHost
			if queue.VHost == nil {
				queue.VHost = new(string)
			}
			if *queue.VHost == "" {
				*queue.VHost = "/"
			}
		}
		if queue.Exchange == "" {
			queue.Exchange = Config.AMQP091Queue.Exchange
		}
		if queue.ExchangeType == "" {
			queue.ExchangeType = Config.AMQP091Queue.ExchangeType
		}
		if queue.Queue == nil {
			queue.Queue = Config.AMQP091Queue.Queue
			if queue.Queue == nil {
				queue.Queue = new(string)
			}
		}
		if queue.RoutingKey == nil {
			queue.RoutingKey = Config.AMQP091Queue.RoutingKey
			if queue.RoutingKey == nil {
				queue.RoutingKey = new(string)
			}
		}
		if queue.Durable == nil {
			queue.Durable = Config.AMQP091Queue.Durable
			if queue.Durable == nil {
				queue.Durable = new(bool)
			}
		}
		if queue.AutoDelete == nil {
			queue.AutoDelete = Config.AMQP091Queue.AutoDelete
			if queue.AutoDelete == nil {
				queue.AutoDelete = new(bool)
			}
		}
	}
	setAMQP091QueueDefaultParams(&Config.AMQP091Queue.OnOpen)
	setAMQP091QueueDefaultParams(&Config.AMQP091Queue.OnMessage)
	setAMQP091QueueDefaultParams(&Config.AMQP091Queue.OnClose)
	if Config.AMQP091Queue.OnOpen.ExchangeType == "" {
		Config.AMQP091Queue.OnOpen.ExchangeType = "direct"
	} else {
		Config.AMQP091Queue.OnOpen.ExchangeType = strings.ToLower(Config.AMQP091Queue.OnOpen.ExchangeType)
		switch Config.AMQP091Queue.OnOpen.ExchangeType {
		case "direct", "fanout", "topic", "headers":
		default:
			slog.Error("Config AMQP091Queue.OnOpen.ExchangeType is invalid")
			os.Exit(1)
		}
	}
	if Config.AMQP091Queue.OnMessage.ExchangeType == "" {
		Config.AMQP091Queue.OnMessage.ExchangeType = "direct"
	} else {
		Config.AMQP091Queue.OnMessage.ExchangeType = strings.ToLower(Config.AMQP091Queue.OnMessage.ExchangeType)
		switch Config.AMQP091Queue.OnMessage.ExchangeType {
		case "direct", "fanout", "topic", "headers":
		default:
			slog.Error("Config AMQP091Queue.OnMessage.ExchangeType is invalid")
			os.Exit(1)
		}
	}
	if Config.AMQP091Queue.OnClose.ExchangeType == "" {
		Config.AMQP091Queue.OnClose.ExchangeType = "direct"
	} else {
		Config.AMQP091Queue.OnClose.ExchangeType = strings.ToLower(Config.AMQP091Queue.OnClose.ExchangeType)
		switch Config.AMQP091Queue.OnClose.ExchangeType {
		case "direct", "fanout", "topic", "headers":
		default:
			slog.Error("Config AMQP091Queue.OnClose.ExchangeType is invalid")
			os.Exit(1)
		}
	}
}
