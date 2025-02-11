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

// 对接网关支持的各种操作接口
package main

import (
	_ "embed"
	"errors"
	_ "github.com/buexplain/netsvr-business-go"
	"html/template"
	"net"
	"net/http"
	"netsvr/pkg/quit"
	"netsvr/test/business/assets"
	"netsvr/test/business/configs"
	"netsvr/test/business/internal/cmd"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/mainSocketManager"
	"netsvr/test/business/internal/netBus"
	"netsvr/test/pkg/protocol"
	"os"
	"strings"
)

func main() {
	//启动html客户端的服务器
	go clientServer()
	mainSocketManager.Init(cmd.EventHandler)
	netBus.Init()
	if mainSocketManager.MainSocketManager.Start() == false {
		log.Logger.Error().Msg("注册到worker服务器失败")
		os.Exit(1)
	}
	log.Logger.Debug().Msg("注册到worker服务器成功")
	//处理关闭信号
	quit.Wg.Add(1)
	go func() {
		defer func() {
			_ = recover()
			quit.Wg.Done()
		}()
		<-quit.Ctx.Done()
		mainSocketManager.MainSocketManager.Close()
	}()
	//开始关闭进程
	select {
	case <-quit.ClosedCh:
		//及时打印关闭进程的日志，避免使用者认为进程无反应，直接强杀进程
		log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("开始关闭business进程")
		//通知所有协程开始退出
		quit.Cancel()
		//等待协程退出
		quit.Wg.Wait()
		log.Logger.Info().Int("pid", os.Getpid()).Str("reason", quit.GetReason()).Msg("关闭business进程成功")
		os.Exit(0)
	}
}

// 输出html客户端
func clientServer() {
	if configs.Config.ClientListenAddress == "" {
		return
	}
	checkIsOpen := func(addr string) bool {
		c, err := net.Dial("tcp", addr)
		if err == nil {
			_ = c.Close()
			return true
		}
		var e *net.OpError
		if errors.As(err, &e) && (strings.Contains(e.Err.Error(), "No connection") || strings.Contains(e.Err.Error(), "connection refused")) {
			return false
		}
		return true
	}
	if checkIsOpen(configs.Config.ClientListenAddress) {
		log.Logger.Info().Msg("地址已被占用: " + configs.Config.ClientListenAddress)
		return
	}
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		t, err := template.New("").Delims("{!", "!}").Parse(assets.GetClientHtml())
		if err != nil {
			log.Logger.Error().Err(err).Msg("模板解析失败")
			return
		}
		data := map[string]interface{}{}
		//注入连接地址
		connUrl := configs.Config.CustomerWsAddress
		data["conn"] = connUrl
		//把所有的命令注入到客户端
		for c, name := range protocol.CmdName {
			data[name] = int(c)
		}
		data["heartbeatMessage"] = string(configs.Config.CustomerHeartbeatMessage)
		err = t.Execute(writer, data)
		if err != nil {
			log.Logger.Error().Msgf("模板输出失败：%s", err)
			return
		}
	})
	log.Logger.Info().Msg("点击访问客户端：http" + ":" + "//" + configs.Config.ClientListenAddress + "/")
	_ = http.ListenAndServe(configs.Config.ClientListenAddress, nil)
}
