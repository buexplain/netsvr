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

// Package worker 客户数据转发模块
// 负责将客户数据转发给business
package worker

import (
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"net"
	"netsvr/configs"
	"netsvr/internal/cmd"
	"netsvr/internal/log"
	"netsvr/internal/worker/manager"
	"netsvr/pkg/quit"
	"os"
	"time"
)

type Server struct {
	listener net.Listener
}

func (r *Server) Start() {
	defer func() {
		log.Logger.Debug().Msg("Worker tcp stop accept")
	}()
	var delay int64 = 0
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if delay == 0 {
					delay = 15
				} else {
					delay *= 2
				}
				if delay > 1000 {
					delay = 1000
				}
				time.Sleep(time.Millisecond * time.Duration(delay))
				continue
			}
			return
		}
		select {
		case <-quit.Ctx.Done():
			//进程即将停止，不再受理新的连接
			_ = conn.Close()
			continue
		default:
			c := manager.NewConnProcessor(conn)
			c.RegisterCmd(netsvrProtocol.Cmd_Register, cmd.Register)
			c.RegisterCmd(netsvrProtocol.Cmd_Unregister, cmd.Unregister)
			c.RegisterCmd(netsvrProtocol.Cmd_Broadcast, cmd.Broadcast)
			c.RegisterCmd(netsvrProtocol.Cmd_InfoUpdate, cmd.InfoUpdate)
			c.RegisterCmd(netsvrProtocol.Cmd_InfoDelete, cmd.InfoDelete)
			c.RegisterCmd(netsvrProtocol.Cmd_ForceOffline, cmd.ForceOffline)
			c.RegisterCmd(netsvrProtocol.Cmd_ForceOfflineGuest, cmd.ForceOfflineGuest)
			c.RegisterCmd(netsvrProtocol.Cmd_Multicast, cmd.Multicast)
			c.RegisterCmd(netsvrProtocol.Cmd_TopicPublish, cmd.TopicPublish)
			c.RegisterCmd(netsvrProtocol.Cmd_SingleCast, cmd.SingleCast)
			c.RegisterCmd(netsvrProtocol.Cmd_TopicSubscribe, cmd.TopicSubscribe)
			c.RegisterCmd(netsvrProtocol.Cmd_TopicUnsubscribe, cmd.TopicUnsubscribe)
			c.RegisterCmd(netsvrProtocol.Cmd_TopicDelete, cmd.TopicDelete)
			c.RegisterCmd(netsvrProtocol.Cmd_UniqIdList, cmd.UniqIdList)
			c.RegisterCmd(netsvrProtocol.Cmd_UniqIdCount, cmd.UniqIdCount)
			c.RegisterCmd(netsvrProtocol.Cmd_TopicUniqIdList, cmd.TopicUniqIdList)
			c.RegisterCmd(netsvrProtocol.Cmd_TopicUniqIdCount, cmd.TopicUniqIdCount)
			c.RegisterCmd(netsvrProtocol.Cmd_TopicCount, cmd.TopicCount)
			c.RegisterCmd(netsvrProtocol.Cmd_TopicList, cmd.TopicList)
			c.RegisterCmd(netsvrProtocol.Cmd_Info, cmd.Info)
			c.RegisterCmd(netsvrProtocol.Cmd_Metrics, cmd.Metrics)
			c.RegisterCmd(netsvrProtocol.Cmd_CheckOnline, cmd.CheckOnline)
			c.RegisterCmd(netsvrProtocol.Cmd_Limit, cmd.Limit)
			//启动三条协程，负责处理命令、读取数据、写入数据、更多的处理命令协程，business在注册的时候可以自定义，要求worker进行开启
			go c.LoopCmd()
			go c.LoopReceive()
			go c.LoopSend()
			quit.Wg.Add(1)
			go func() {
				defer func() {
					_ = recover()
					quit.Wg.Done()
				}()
				select {
				case <-quit.Ctx.Done():
					c.ForceClose()
					return
				case <-c.GetCloseCh():
					return
				}
			}()
		}
	}
}

var server *Server

func Start() {
	listen, err := net.Listen("tcp", configs.Config.Worker.ListenAddress)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Worker tcp start failed")
		time.Sleep(time.Millisecond * 100)
		os.Exit(1)
		return
	}
	server = &Server{
		listener: listen,
	}
	log.Logger.Info().Msg("Worker tcp start")
	server.Start()
}

func Shutdown() {
	err := server.listener.Close()
	if err != nil {
		log.Logger.Error().Err(err).Msg("Worker tcp grace shutdown failed")
		return
	}
	log.Logger.Info().Msg("Worker tcp grace shutdown")
}
