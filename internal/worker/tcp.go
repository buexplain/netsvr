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

package worker

import (
	"errors"
	"net"
	"netsvr/configs"
	"netsvr/internal/log"
	"netsvr/pkg/quit"
	"os"
	"time"
)

type Server struct {
	listener net.Listener
}

func (r *Server) Start() {
	defer func() {
		log.Logger.Debug().Int("pid", os.Getpid()).Msg("Worker tcp stop accept")
	}()
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			select {
			case <-quit.ClosedCh:
				//进程收到停止信号，不再受理新的连接
				return
			default:
				var ne net.Error
				if errors.As(err, &ne) && ne.Temporary() {
					if tempDelay == 0 {
						tempDelay = 5 * time.Millisecond
					} else {
						tempDelay *= 2
					}
					if maxDelay := 1 * time.Second; tempDelay > maxDelay {
						tempDelay = maxDelay
					}
					log.Logger.Error().Msgf("Worker tcp accept error: %v; retrying in %v", err, tempDelay)
					time.Sleep(tempDelay)
					continue
				}
			}
			return
		}
		select {
		case <-quit.ClosedCh:
			//进程收到停止信号，不再受理新的连接
			_ = conn.Close()
			continue
		default:
			//启动一个协程处理连接
			go process(newConn(conn))
		}
	}
}

var server *Server

func Start() {
	listen, err := net.Listen("tcp", configs.Config.Worker.ListenAddress)
	if err != nil {
		log.Logger.Error().Int("pid", os.Getpid()).Err(err).Msg("Worker tcp start failed")
		time.Sleep(time.Millisecond * 100)
		os.Exit(1)
		return
	}
	server = &Server{
		listener: listen,
	}
	log.Logger.Info().Int("pid", os.Getpid()).Msgf("Worker tcp start tcp://%s", configs.Config.Worker.ListenAddress)
	server.Start()
}

func Shutdown() {
	err := server.listener.Close()
	if err != nil {
		log.Logger.Error().Int("pid", os.Getpid()).Err(err).Msg("Worker tcp shutdown failed")
		return
	}
	log.Logger.Info().Int("pid", os.Getpid()).Msg("Worker tcp shutdown")
}
