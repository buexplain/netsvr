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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v2/netsvr"
	"html/template"
	"net"
	"net/http"
	"netsvr/pkg/quit"
	"netsvr/test/business/configs"
	"netsvr/test/business/internal/cmd"
	"netsvr/test/business/internal/connProcessor"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/web"
	"netsvr/test/pkg/protocol"
	"netsvr/test/pkg/utils"
	"os"
	"strconv"
	"strings"
)

func main() {
	//初始化连接池
	utils.InitRequestNetSvrPool(configs.Config.ProcessCmdGoroutineNum, configs.Config.WorkerListenAddress)
	//连接到网关的worker服务器
	conn, err := net.Dial("tcp", configs.Config.WorkerListenAddress)
	if err != nil {
		log.Logger.Error().Msgf("连接服务端失败，%v", err)
		os.Exit(1)
	}
	//启动html客户端的服务器
	go clientServer()
	processor := connProcessor.NewConnProcessor(conn, configs.Config.WorkerId, configs.Config.ServerId)
	//注册到worker
	if err = processor.RegisterWorker(uint32(configs.Config.ProcessCmdGoroutineNum)); err != nil {
		log.Logger.Error().Int32("workerId", processor.GetWorkerId()).Err(err).Msg("注册到worker服务器失败")
		os.Exit(1)
	}
	log.Logger.Debug().Int32("workerId", processor.GetWorkerId()).Msg("注册到worker服务器成功")
	//注册各种回调函数
	cmd.CheckOnline.Init(processor)
	cmd.Broadcast.Init(processor)
	cmd.Multicast.Init(processor)
	cmd.SingleCast.Init(processor)
	cmd.SingleCastBulk.Init(processor)
	cmd.ConnSwitch.Init(processor)
	cmd.Sign.Init(processor)
	cmd.ForceOffline.Init(processor)
	cmd.ForceOfflineGuest.Init(processor)
	cmd.Topic.Init(processor)
	cmd.UniqId.Init(processor)
	cmd.Metrics.Init(processor)
	cmd.Limit.Init(processor)
	//心跳
	go processor.LoopHeartbeat()
	//循环处理worker发来的指令
	for i := 0; i < configs.Config.ProcessCmdGoroutineNum; i++ {
		//添加到进程结束时的等待中，这样客户发来的数据都会被处理完毕
		quit.Wg.Add(1)
		go processor.LoopCmd()
	}
	//循环写
	go processor.LoopSend()
	//循环读
	go processor.LoopReceive()
	//处理关闭信号
	quit.Wg.Add(1)
	go func() {
		defer func() {
			_ = recover()
			quit.Wg.Done()
		}()
		select {
		case <-processor.GetCloseCh():
			return
		case <-quit.Ctx.Done():
			//取消注册
			processor.UnregisterWorker()
			//优雅的强制关闭
			processor.ForceClose()
		}
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
		if e, ok := err.(*net.OpError); ok && strings.Contains(e.Err.Error(), "No connection") {
			return false
		}
		return true
	}
	if checkIsOpen(configs.Config.ClientListenAddress) {
		log.Logger.Info().Msg("地址已被占用: " + configs.Config.ClientListenAddress)
		return
	}
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		t, err := template.New("").Delims("{!", "!}").Parse(web.Client)
		if err != nil {
			log.Logger.Error().Err(err).Msg("模板解析失败")
			return
		}
		data := map[string]interface{}{}
		//注入连接地址
		connUrl := configs.Config.CustomerWsAddress
		if configs.Config.ConnOpenCustomUniqIdKey != "" {
			resp := &netsvrProtocol.ConnOpenCustomUniqIdTokenResp{}
			utils.RequestNetSvr(nil, netsvrProtocol.Cmd_ConnOpenCustomUniqIdToken, resp)
			if strings.Contains(connUrl, "?") {
				connUrl = connUrl + "&" + configs.Config.ConnOpenCustomUniqIdKey + "=" + resp.UniqId + "&token=" + resp.Token
			} else {
				connUrl = connUrl + "?" + configs.Config.ConnOpenCustomUniqIdKey + "=" + resp.UniqId + "&token=" + resp.Token
			}
		}
		data["conn"] = connUrl
		//把所有的命令注入到客户端
		for c, name := range protocol.CmdName {
			data[name] = int(c)
		}
		data["pingMessage"] = string(netsvrProtocol.PingMessage)
		data["pongMessage"] = string(netsvrProtocol.PongMessage)
		//如果workerId不够三位数，则补上0
		workerId := strconv.Itoa(int(configs.Config.WorkerId))
		for i := len(workerId); i < 3; i++ {
			workerId = "0" + workerId
		}
		data["workerId"] = workerId
		err = t.Execute(writer, data)
		if err != nil {
			log.Logger.Error().Msgf("模板输出失败：%s", err)
			return
		}
	})
	log.Logger.Info().Msg("点击访问客户端：http" + ":" + "//" + configs.Config.ClientListenAddress + "/")
	_ = http.ListenAndServe(configs.Config.ClientListenAddress, nil)
}
