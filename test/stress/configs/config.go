/**
* Copyright 2022 buexplain@qq.com
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
)

type config struct {
	//日志级别 debug、info、warn、error
	LogLevel string
	//customer服务的websocket连接地址
	CustomerWsAddress string
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
		//订阅、取消订阅、发布的间隔
		MessageInterval int
		//发送的消息大小
		MessageLen int
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
}
