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

type config struct {
	//日志级别 debug、info、warn、error
	LogLevel string
	//worker服务的监听地址
	WorkerListenAddress string
	//customer服务的websocket连接地址
	ClientListenAddress string
	//输出客户端界面的http服务的监听地址
	CustomerWsAddress string
	//让worker为我开启n条协程来处理我的请求
	ProcessCmdGoroutineNum int
	//客户端websocket发送的心跳消息
	CustomerHeartbeatMessage BytesConfigItem
	//business进程向网关的worker服务器发送的心跳消息
	WorkerHeartbeatMessage BytesConfigItem
	//伪造登录时，伪造客户业务系统的唯一id的初始值，如果启动多个business进程，则需要区分每个business进程的起始值，最大值为uint32
	ForgeCustomerIdInitVal uint32
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
	if Config.ProcessCmdGoroutineNum <= 0 {
		Config.ProcessCmdGoroutineNum = 1
	}
}
