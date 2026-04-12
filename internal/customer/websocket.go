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

package customer

import (
	"bytes"
	"context"
	"errors"
	gTimer "github.com/antlabs/timer"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/gobwas/ws"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"github.com/rs/zerolog"
	"io"
	"math"
	"net"
	"net/http"
	"netsvr/configs"
	"netsvr/internal/customer/binder"
	"netsvr/internal/customer/callback"
	"netsvr/internal/customer/info"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/customer/uniqIdGen"
	"netsvr/internal/limit"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/objPool"
	"netsvr/internal/timer"
	"netsvr/internal/worker"
	"netsvr/internal/wsServer"
	"netsvr/pkg/quit"
	"os"
	"runtime"
	"strings"
	"time"
)

var server *wsServer.Server

func Start() {
	server = &wsServer.Server{
		OnUpgradeCheck: func(req *http.Request) *http.Response {
			// 检查请求路径
			if req.RequestURI != configs.Config.Customer.HandlePattern {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					ProtoMajor: 1,
					ProtoMinor: 1,
					Body:       io.NopCloser(strings.NewReader(http.StatusText(http.StatusNotFound))),
				}
			}
			// 检查请求源
			if len(configs.Config.Customer.AllowOrigin) > 0 {
				origin := req.Header.Get("Origin")
				ok := false
				for _, v := range configs.Config.Customer.AllowOrigin {
					if strings.Contains(origin, v) {
						ok = true
						break
					}
				}
				if !ok {
					return &http.Response{
						StatusCode: http.StatusForbidden,
						ProtoMajor: 1,
						ProtoMinor: 1,
						Body:       io.NopCloser(strings.NewReader(http.StatusText(http.StatusForbidden))),
					}
				}
			}
			return nil
		},
		OnWebsocketOpen: func(wsCodec *wsServer.Codec, req *http.Request) (ws.StatusCode, error) {
			//进程即将关闭，不再受理新的连接
			select {
			case <-quit.Ctx.Done():
				return ws.StatusCode(1012), errors.New("service restart")
			default:
			}
			//检查在线人数
			if server.OnlineNum() >= configs.Config.Customer.MaxOnlineNum {
				return ws.StatusCode(1013), errors.New("open rate limited, try again later")
			}
			//限流检查
			if limit.Manager.Allow(netsvrProtocol.Event_OnOpen) == false {
				//统计连接打开的限流次数
				metrics.Registry[metrics.ItemOpenRateLimitCount].Meter.Mark(1)
				//触发了限流要打错误日志告警，因为这个时候有可能是因为客户消息太多了，网关的business进程处理不过来
				log.Logger.Error().
					Str("rawQuery", req.URL.RawQuery).
					Str("xForwardedFor", req.Header.Get("X-Forwarded-For")).
					Str("xRealIp", req.Header.Get("X-Real-IP")).
					Str("remoteAddr", req.RemoteAddr).
					Str("customerListenAddress", configs.Config.Customer.ListenAddress).
					Msg("open rate limited")
				return ws.StatusCode(1013), errors.New("open rate limited, try again later")
			}
			err := goroutine.DefaultWorkerPool.Submit(func() {
				defer func() {
					if err := recover(); err != nil {
						log.Logger.Error().
							Stack().Err(nil).
							Type("recoverType", err).
							Interface("recover", err).
							Msg("OnWebsocketOpen failed")
					}
				}()
				//当闭包开始执行的时候，conn可能已经关闭了
				if wsCodec.IsClosed() {
					return
				}
				remoteAddr, _, _ := net.SplitHostPort(wsCodec.RemoteAddr().String())
				uniqId := uniqIdGen.New()
				co := objPool.ConnOpen.Get()
				defer objPool.ConnOpen.Put(co)
				co.UniqId = uniqId
				co.RawQuery = req.URL.RawQuery
				co.XForwardedFor = req.Header.Get("X-Forwarded-For")
				co.XRealIp = req.Header.Get("X-Real-IP")
				co.RemoteAddr = remoteAddr
				var connOpenResp *netsvrProtocol.ConnOpenResp
				var err error
				if configs.Config.Customer.OnOpenCallbackApi != "" {
					connOpenResp, err = callback.OnOpen(co)
					//回调函数回来后，连接可能已经关闭了
					if wsCodec.IsClosed() {
						return
					}
					//回调错误，写入关闭帧
					if err != nil {
						WriteClose(wsCodec, ws.StatusInternalServerError, err)
						return
					}
					//不允许连接，写入关闭帧
					if connOpenResp.Allow == false {
						if len(connOpenResp.Data) > 0 {
							WriteMessage(wsCodec, configs.Config.Customer.SendMessageType, connOpenResp.Data)
						}
						WriteClose(wsCodec, ws.StatusPolicyViolation, errors.New("unauthorized"))
						return
					}
				}
				//统计客户连接的打开次数
				metrics.Registry[metrics.ItemCustomerConnOpenCount].Meter.Mark(1)
				session := info.New(uniqId)
				//设置回调返回的数据
				if connOpenResp != nil {
					if connOpenResp.NewSession != "" {
						session.SetSession(connOpenResp.NewSession)
					}
					if connOpenResp.NewCustomerId != "" {
						session.SetCustomerId(connOpenResp.NewCustomerId)
					}
					if len(connOpenResp.NewTopics) > 0 {
						session.SubscribeTopics(connOpenResp.NewTopics)
					}
				}
				//让连接持有session info对象
				wsCodec.SetSession(session)
				//添加到连接管理器
				manager.Manager.Set(uniqId, wsCodec)
				//构建回调返回的关系，并将数据下发给客户端
				if connOpenResp != nil {
					if connOpenResp.NewCustomerId != "" {
						binder.Binder.SetRelation(connOpenResp.NewCustomerId, wsCodec)
					}
					if len(connOpenResp.NewTopics) > 0 {
						topic.Topic.SetRelation(connOpenResp.NewTopics, wsCodec)
					}
					if len(connOpenResp.Data) > 0 {
						WriteMessage(wsCodec, configs.Config.Customer.SendMessageType, connOpenResp.Data)
					}
				}
				//添加心跳检查
				var tNode gTimer.TimeNoder
				tNode = timer.Timer.ScheduleFunc(configs.Config.Customer.HeartbeatInterval, func() {
					//检测连接是否已经关闭
					if wsCodec.IsClosed() {
						if tNode != nil {
							tNode.Stop()
						}
						return
					}
					//检查最后活跃时间
					lastActiveTime := session.GetLastActiveTimeOnSafe()
					if uint32(time.Now().Unix())-lastActiveTime > configs.Config.Customer.HeartbeatIntervalSecond {
						if tNode != nil {
							tNode.Stop()
						}
						WriteClose(wsCodec, ws.StatusGoingAway, errors.New("heartbeat timeout"))
					}
				})
				//需要将客户端连接打开的信息转发给business进程
				if currentWorker := worker.Manager.Get(netsvrProtocol.Event_OnOpen); currentWorker != nil {
					if sendSize := currentWorker.Send(co, netsvrProtocol.Cmd_ConnOpen); sendSize > 0 {
						//统计转发到business的次数与字节数
						metrics.Registry[metrics.ItemCustomerTransferCount].Meter.Mark(1)
						metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(sendSize))
					}
				}
			})
			if err != nil {
				return ws.StatusInternalServerError, err
			}
			return ws.StatusCode(0), nil
		},
		OnWebsocketClose: func(wsCodec *wsServer.Codec) {
			session, ok := wsCodec.GetSession().(*info.Info)
			if !ok {
				//open阶段还未设置session，则直接返回
				return
			}
			fn := func() {
				defer func() {
					if err := recover(); err != nil {
						log.Logger.Error().
							Stack().Err(nil).
							Type("recoverType", err).
							Interface("recover", err).
							Msg("OnWebsocketClose failed")
					}
				}()
				session.RLock()
				uniqId, customerId, customerSession, topics := session.Snapshot()
				//释放锁
				session.RUnlock()
				//统计客户连接的关闭次数
				metrics.Registry[metrics.ItemCustomerConnCloseCount].Meter.Mark(1)
				//从连接管理器中删除
				manager.Manager.Del(uniqId)
				//解除uniqId与customerId的关系
				if customerId != "" {
					binder.Binder.DelRelation(customerId, wsCodec)
				}
				//删除订阅关系
				topic.Topic.DelRelationBySlice(topics, wsCodec)
				cl := objPool.ConnClose.Get()
				defer objPool.ConnClose.Put(cl)
				cl.UniqId = uniqId
				cl.CustomerId = customerId
				cl.Session = customerSession
				cl.Topics = topics
				if configs.Config.Customer.OnCloseCallbackApi != "" {
					//如果网关程序仅作推送消息的场景，则business进程是不存在的，所以这里需要回调，方便此场景下处理连接关闭的事件
					callback.OnClose(cl)
				}
				//将连接关闭的消息转发给business进程
				if currentWorker := worker.Manager.Get(netsvrProtocol.Event_OnClose); currentWorker != nil {
					if sendSize := currentWorker.Send(cl, netsvrProtocol.Cmd_ConnClose); sendSize > 0 {
						//统计客户数据转发到worker的次数与字节数情况
						metrics.Registry[metrics.ItemCustomerTransferCount].Meter.Mark(1)
						metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(sendSize))
					}
				}
			}
			if err := goroutine.DefaultWorkerPool.Submit(fn); err != nil {
				//提交异步任务失败，则使用goroutine执行，保证已经关闭的连接能正常清理
				go fn()
				log.Logger.Warn().Err(err).Msg("OnWebsocketClose submit to worker pool failed")
			}
		},
		OnWebsocketPing: func(wsCodec *wsServer.Codec) {
			session, ok := wsCodec.GetSession().(*info.Info)
			if !ok {
				//open阶段还未设置session，则直接返回
				return
			}
			//更新最后活跃时间
			session.UpdateLastActiveTimeOnSafe()
			metrics.Registry[metrics.ItemCustomerHeartbeatCount].Meter.Mark(1)
		},
		OnWebsocketMessage: func(wsCodec *wsServer.Codec, messageType ws.OpCode, data []byte) {
			session, ok := wsCodec.GetSession().(*info.Info)
			if !ok {
				//open阶段还未设置session，则直接返回
				return
			}
			//更新最后活跃时间
			session.UpdateLastActiveTimeOnSafe()
			//判断是否是心跳包
			if bytes.Equal(data, configs.Config.Customer.HeartbeatMessage) {
				metrics.Registry[metrics.ItemCustomerHeartbeatCount].Meter.Mark(1)
				return
			}
			currentWorker := worker.Manager.Get(netsvrProtocol.Event_OnMessage)
			if currentWorker == nil {
				return
			}
			//获取session中的数据
			session.RLock()
			allow := session.Allow()
			uniqId, customerId, customerSession, topics := session.Snapshot()
			session.RUnlock()
			//限制数据包大小，溢出限制大小，直接丢弃该数据
			if len(data) > configs.Config.Customer.ReceivePackLimit {
				WriteClose(wsCodec, ws.StatusMessageTooBig, errors.New("message too big"))
				//打日志，方便客户端排查问题
				log.Logger.Info().
					Str("uniqId", uniqId).
					Str("customerId", customerId).
					Str("customerSession", customerSession).
					Str("remoteAddr", wsCodec.RemoteAddr().String()).
					Str("customerListenAddress", configs.Config.Customer.ListenAddress).
					Msg("message too large")
				return
			}
			//连接限流检查
			if allow == false {
				//统计连接消息限流次数
				metrics.Registry[metrics.ItemConnectionMessageRateLimitCount].Meter.Mark(1)
				//触发了限流要打错误日志告警，因为这个时候有可能是被人攻击，或者是客户端的业务逻辑问题，导致请求太多
				formatCustomerData(data, log.Logger.Error()).
					Str("uniqId", uniqId).
					Str("customerId", customerId).
					Str("customerSession", customerSession).
					Str("remoteAddr", wsCodec.RemoteAddr().String()).
					Str("customerListenAddress", configs.Config.Customer.ListenAddress).
					Msg("connection message rate limited")
				return
			}
			//全局限流检查
			if limit.Manager.Allow(netsvrProtocol.Event_OnMessage) == false {
				//统计客户消息限流次数
				metrics.Registry[metrics.ItemMessageRateLimitCount].Meter.Mark(1)
				//触发了限流要打错误日志告警，因为这个时候有可能是因为客户消息太多了，网关的business进程处理不过来
				formatCustomerData(data, log.Logger.Error()).
					Str("uniqId", uniqId).
					Str("customerId", customerId).
					Str("customerSession", customerSession).
					Str("remoteAddr", wsCodec.RemoteAddr().String()).
					Str("customerListenAddress", configs.Config.Customer.ListenAddress).
					Msg("message rate limited")
				return
			}
			//记录所有请求日志
			if configs.Config.LogLevel == "debug" {
				formatCustomerData(data, log.Logger.Debug()).
					Str("uniqId", uniqId).
					Str("customerId", customerId).
					Str("customerSession", customerSession).
					Str("remoteAddr", wsCodec.RemoteAddr().String()).
					Str("customerListenAddress", configs.Config.Customer.ListenAddress).
					Send()
			}
			fn := func() {
				defer func() {
					if err := recover(); err != nil {
						log.Logger.Error().
							Stack().Err(nil).
							Type("recoverType", err).
							Interface("recover", err).
							Msg("OnWebsocketMessage failed")
					}
				}()
				//编码数据成business需要的格式
				tf := objPool.Transfer.Get()
				tf.UniqId = uniqId
				tf.CustomerId = customerId
				tf.Session = customerSession
				tf.Topics = topics
				tf.Data = data
				defer objPool.Transfer.Put(tf)
				//转发数据到business
				if sendSize := currentWorker.Send(tf, netsvrProtocol.Cmd_Transfer); sendSize > 0 {
					//统计转发到business的次数与字节数
					metrics.Registry[metrics.ItemCustomerTransferCount].Meter.Mark(1)
					metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(sendSize))
				}
			}
			err := goroutine.DefaultWorkerPool.Submit(fn)
			if err != nil {
				//提交异步任务失败，则使用goroutine执行，保证数据发送成功
				go fn()
				log.Logger.Warn().Err(err).Msg("OnWebsocketMessage submit to worker pool failed")
			}
		},
	}
	go func() {
		time.Sleep(time.Second * 2)
		log.Logger.Info().Int("pid", os.Getpid()).Msgf(
			"Customer websocket start ws://%s%s",
			configs.Config.Customer.ListenAddress,
			configs.Config.Customer.HandlePattern,
		)
	}()
	var numEventLoop int
	if configs.Config.Customer.Multicore == 0 {
		numEventLoop = 1
	} else {
		numEventLoop = int(math.Ceil(float64(runtime.NumCPU()) * (float64(configs.Config.Customer.Multicore) / 100)))
	}
	err := gnet.Run(
		server,
		"tcp://"+configs.Config.Customer.ListenAddress,
		gnet.WithLogger(log.NewLoggingSubstitute(&log.Logger)),
		gnet.WithNumEventLoop(numEventLoop),
		gnet.WithReusePort(true),
		gnet.WithReuseAddr(true),
		gnet.WithLockOSThread(true),
		gnet.WithLoadBalancing(gnet.LeastConnections),
		gnet.WithLogLevel(logging.InfoLevel),
	)
	if err != nil {
		log.Logger.Error().Err(err).Int("pid", os.Getpid()).Msg("Customer websocket start failed")
		time.Sleep(time.Millisecond * 100)
		os.Exit(1)
	}
}

func StartAutobahn() {
	server = &wsServer.Server{
		OnUpgradeCheck: func(req *http.Request) *http.Response {
			return nil
		},
		OnWebsocketOpen: func(wsCodec *wsServer.Codec, req *http.Request) (ws.StatusCode, error) {
			return ws.StatusCode(0), nil
		},
		OnWebsocketClose: func(wsCodec *wsServer.Codec) {
		},
		OnWebsocketPing: func(wsCodec *wsServer.Codec) {
		},
		OnWebsocketMessage: func(wsCodec *wsServer.Codec, messageType ws.OpCode, data []byte) {
			fn := func() {
				//两个写入方法都要测试一次
				//NewMessage(messageType, data).WriteTo(conn)
				WriteMessage(wsCodec, messageType, data)
			}
			err := goroutine.DefaultWorkerPool.Submit(fn)
			if err != nil {
				//提交异步任务失败，则使用goroutine执行，保证数据发送成功
				go fn()
			}
		},
	}
	go func() {
		time.Sleep(time.Second * 2)
		log.Logger.Info().Int("pid", os.Getpid()).Msgf(
			"Autobahn websocket start ws://%s%s",
			configs.Config.Customer.ListenAddress,
			configs.Config.Customer.HandlePattern,
		)
	}()
	var numEventLoop int
	if configs.Config.Customer.Multicore == 0 {
		numEventLoop = 1
	} else {
		numEventLoop = int(math.Ceil(float64(runtime.NumCPU()) * (float64(configs.Config.Customer.Multicore) / 100)))
	}
	err := gnet.Run(
		server,
		"tcp://"+configs.Config.Customer.ListenAddress,
		gnet.WithLogger(log.NewLoggingSubstitute(&log.Logger)),
		gnet.WithNumEventLoop(numEventLoop),
		gnet.WithReusePort(true),
		gnet.WithReuseAddr(true),
		gnet.WithLockOSThread(true),
		gnet.WithLoadBalancing(gnet.LeastConnections),
		gnet.WithLogLevel(logging.InfoLevel),
	)
	if err != nil {
		log.Logger.Error().Err(err).Int("pid", os.Getpid()).Msg("Autobahn websocket start failed")
		time.Sleep(time.Millisecond * 100)
		os.Exit(1)
	}
}

func Shutdown() {
	err := server.Shutdown(context.Background())
	if err != nil {
		log.Logger.Error().Int("pid", os.Getpid()).Err(err).Msg("Customer websocket shutdown failed")
		return
	}
	log.Logger.Info().Int("pid", os.Getpid()).Msg("Customer websocket shutdown")
}

func formatCustomerData(customerData []byte, event *zerolog.Event) *zerolog.Event {
	if configs.Config.Customer.SendMessageType == ws.OpText {
		return event.Str("customerData", string(customerData))
	}
	return event.Hex("customerDataHex", customerData)
}
