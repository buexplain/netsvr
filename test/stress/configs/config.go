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
	"github.com/lesismal/nbio/logging"
	"github.com/rs/zerolog"
	"netsvr/pkg/wd"
	"os"
	"path/filepath"
	"strconv"
)

type config struct {
	//日志级别 debug、info、warn、error
	LogLevel string
	//customer服务的websocket连接地址
	CustomerWsAddress string
	//表示消息被哪个workerId的业务进程处理
	WorkerId      int
	WorkerIdBytes []byte
	//心跳间隔秒数
	Heartbeat int
	//是否并发初始化
	Concurrent bool
	//当所有连接建立完毕后的持续时间，单位秒
	Suspend int
	//沉默的大多数
	Silent struct {
		Enable bool
		Step   []Step
	}
	//疯狂的登录登出
	Sign struct {
		Enable bool
		//发消息的间隔，单位秒
		MessageInterval int
		//阶段式发起连接
		Step []Step
	}
	//疯狂单播
	SingleCast struct {
		Enable bool
		//发送的消息大小
		MessageLen int
		//发消息的间隔，单位秒
		MessageInterval int
		//阶段式发起连接
		Step []Step
	}
	//疯狂组播
	Multicast struct {
		Enable bool
		//发送的消息大小
		MessageLen int
		//一组包含的uniqId数量
		UniqIdNum int
		//发消息的间隔，单位秒
		MessageInterval int
		//阶段式发起连接
		Step []Step
	}
	//疯狂订阅、取消订阅、发布
	Topic struct {
		Enable bool
		//每个连接的备选主题数量，该值是随机产生的，该值越大，则一个主题会被多个连接订阅的概率多变的很小
		AlternateTopicNum int
		//主题字符串的长度，4有1413720种排列 3有42840种排列，2有1260种排列，该值越大，则一个主题会被多个连接订阅的概率多变的很小
		AlternateTopicLen int
		Subscribe         struct {
			//运行模式 0 延长多少秒执行，1 间隔多少秒执行一次
			Mode int
			//延长或间隔的秒数
			ModeSecond int
			//每次订阅的主题数量，该值是轮询AlternateTopicNum得到的，第零个主题一定会被订阅
			TopicNum int
		}
		Unsubscribe struct {
			//运行模式 0 延长多少秒执行，1 间隔多少秒执行一次
			Mode int
			//延长或间隔的秒数
			ModeSecond int
			//每次取消订阅的主题数量，该值是轮询AlternateTopicNum得到的，但是不会取消连接订阅的第零个主题
			TopicNum int
		}
		Publish struct {
			//运行模式 0 延长多少秒执行，1 间隔多少秒执行一次
			Mode int
			//延长或间隔的秒数
			ModeSecond int
			//每次发布消息的主题数量，该值是轮询AlternateTopicNum得到的，但是连接订阅的第零个主题是一定会被用于消息发布的
			TopicNum int
			//发布数据的大小
			MessageLen int
		}
		//阶段式发起连接
		Step []Step
	}
	//疯狂广播
	Broadcast struct {
		Enable bool
		//发送的消息大小
		MessageLen int
		//发消息的间隔，单位秒
		MessageInterval int
		//阶段式发起连接
		Step []Step
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

// ModeSchedule 调度方式
const ModeSchedule = 1
const ModeAfter = 0

type Step struct {
	//连接数
	ConnNum int
	//每秒发起的连接数量
	ConnectNum int
	//连接完成后，暂停时间，单位秒，该时间后进入下一个step
	Suspend int
}

var Config *config

func init() {
	var configFile string
	flag.StringVar(&configFile, "config", filepath.Join(wd.RootPath, "configs/stress.toml"), "Set stress.toml file")
	flag.Parse()
	//读取配置文件
	c, err := os.ReadFile(configFile)
	if err != nil {
		logging.Error("Read stress.toml failed：%s", err)
		os.Exit(1)
	}
	//解析配置文件到对象
	Config = new(config)
	if metaData, err := toml.Decode(string(c), Config); err != nil {
		logging.Error("Parse stress.toml failed：%s", err)
		os.Exit(1)
	} else if metaData.IsDefined("ShutdownWaitTime") {
		logging.Error("Process working directory is error: %s", wd.RootPath)
		os.Exit(1)
	}
	if Config.Suspend <= 0 {
		Config.Suspend = 60
	}
	//如果workerId不够三位数，则补上0
	s := strconv.Itoa(Config.WorkerId)
	for i := len(s); i < 3; i++ {
		s = "0" + s
	}
	Config.WorkerIdBytes = []byte(s)
}
