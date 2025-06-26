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

// Package customer 客户连接维持模块
// 负责承载客户的websocket连接、维护客户的订阅、保持客户连接的session数据
package customer

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"github.com/lesismal/llib/std/crypto/tls"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"github.com/rs/zerolog"
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
	"netsvr/internal/utils"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/quit"
	"os"
	"strings"
	"time"
)

var server *nbhttp.Server
var upgrade = (func() *websocket.Upgrader {
	upgrade := websocket.NewUpgrader()
	upgrade.KeepaliveTime = configs.Config.Customer.ReadDeadline
	upgrade.HandshakeTimeout = configs.Config.Customer.SendDeadline * 2
	upgrade.CheckOrigin = checkOrigin
	upgrade.SetPingHandler(pingMessageHandler)
	upgrade.OnOpen(nil)
	upgrade.OnClose(onClose)
	upgrade.OnMessage(onMessage)
	//处理websocket子协议
	upgrade.Subprotocols = nil
	if configs.Config.Customer.CompressionLevel == 0 {
		upgrade.EnableCompression(false)
	} else {
		upgrade.EnableCompression(true)
		_ = upgrade.SetCompressionLevel(configs.Config.Customer.CompressionLevel)
	}
	return upgrade
})()

// OpenRateLimit 不能轻易改变这些常量的值，因为websocket客户端代码极有可能判断这些字符串做出下一步处理，改变这些字符会导致这些客户端的代码失效
const OpenRateLimit = "Open rate limited"
const MessageRateLimit = "Message rate limited"
const ConnectionMessageRateLimit = "Connection message rate limited"
const ServiceShutdown = "Service shutdown"
const MessageTooLarge = "Message too large"

func Start() {
	var tlsConfig *tls.Config
	var connAddr string
	if configs.Config.Customer.TLSKey != "" || configs.Config.Customer.TLSCert != "" {
		cert, err := tls.LoadX509KeyPair(configs.Config.Customer.TLSCert, configs.Config.Customer.TLSKey)
		if err != nil {
			log.Logger.Error().Int("pid", os.Getpid()).Err(err).Msg("Customer websocket tls.LoadX509KeyPair failed")
			os.Exit(1)
		}
		tlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
	}
	config := nbhttp.Config{
		Network:           "tcp",
		Name:              "customer",
		MaxLoad:           configs.Config.Customer.MaxOnlineNum,
		IOMod:             configs.Config.Customer.IOMod,
		MaxBlockingOnline: configs.Config.Customer.MaxBlockingOnline,
	}
	if tlsConfig == nil {
		config.Addrs = []string{configs.Config.Customer.ListenAddress}
		connAddr = "ws://" + configs.Config.Customer.ListenAddress + configs.Config.Customer.HandlePattern
	} else {
		config.AddrsTLS = []string{configs.Config.Customer.ListenAddress}
		config.TLSConfig = tlsConfig
		connAddr = "wss://" + configs.Config.Customer.ListenAddress + configs.Config.Customer.HandlePattern
	}
	mux := &http.ServeMux{}
	mux.HandleFunc(configs.Config.Customer.HandlePattern, onWebsocket)
	config.Handler = mux
	server = nbhttp.NewServer(config)
	err := server.Start()
	if err != nil {
		log.Logger.Error().Err(err).Int("pid", os.Getpid()).Msg("Customer websocket start failed")
		time.Sleep(time.Millisecond * 100)
		os.Exit(1)
		return
	}
	log.Logger.Info().Int("pid", os.Getpid()).Msgf("Customer websocket start %s", connAddr)
}

func Shutdown() {
	err := server.Shutdown(context.Background())
	if err != nil {
		log.Logger.Error().Int("pid", os.Getpid()).Err(err).Msg("Customer websocket grace shutdown failed")
		return
	}
	log.Logger.Info().Int("pid", os.Getpid()).Msg("Customer websocket grace shutdown")
}

func onWebsocket(w http.ResponseWriter, r *http.Request) {
	select {
	case <-quit.Ctx.Done():
		//进程即将关闭，不再受理新的连接
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(utils.StrToReadOnlyBytes(ServiceShutdown))
		return
	default:
	}
	//检测是否是websocket连接，如果不是，则视为负载层发送的健康检查请求
	if r.Header.Get("Upgrade") == "" {
		http.Error(w, "Hello netsvr", http.StatusOK)
		return
	}
	//获取ws子协议
	subProtocols := utils.ParseSubProtocols(r)
	var responseHeader http.Header
	if subProtocols != nil {
		//返回任意一个子协议，保证连接升级成功
		responseHeader = http.Header{}
		responseHeader["Sec-Websocket-Protocol"] = subProtocols[0:1]
	}
	//获取部分header信息
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	xRealIp := r.Header.Get("X-Real-IP")
	remoteAddr, _, _ := net.SplitHostPort(r.RemoteAddr)
	//限流检查
	if limit.Manager.Allow(netsvrProtocol.Event_OnOpen) == false {
		//统计连接打开的限流次数
		metrics.Registry[metrics.ItemOpenRateLimitCount].Meter.Mark(1)
		//触发了限流要打错误日志告警，因为这个时候有可能是因为客户消息太多了，网关的business进程处理不过来
		log.Logger.Error().
			Str("rawQuery", r.URL.RawQuery).
			Strs("subProtocols", subProtocols).
			Str("xForwardedFor", xForwardedFor).
			Str("xRealIp", xRealIp).
			Str("remoteAddr", r.RemoteAddr).
			Str("customerListenAddress", configs.Config.Customer.ListenAddress).
			Msg(OpenRateLimit)
		return
	}
	//生成唯一id
	uniqId := uniqIdGen.New()
	//开始升级
	wsConn, err := upgrade.Upgrade(w, r, responseHeader)
	if err != nil {
		log.Logger.Error().Err(err).
			Str("rawQuery", r.URL.RawQuery).
			Strs("subProtocols", subProtocols).
			Str("xForwardedFor", xForwardedFor).
			Str("xRealIp", xRealIp).
			Str("remoteAddr", r.RemoteAddr).
			Str("customerListenAddress", configs.Config.Customer.ListenAddress).
			Msg("Customer websocket upgrade failed")
		return
	}
	//调用回调函数
	var onOpenResp *callback.OnOpenResp
	if configs.Config.Customer.OnOpenCallbackApi != "" {
		onOpenReq := &callback.OnOpenReq{
			MessageType:   int8(configs.Config.Customer.SendMessageType),
			RawQuery:      r.URL.RawQuery,
			RemoteAddr:    remoteAddr,
			SubProtocols:  subProtocols,
			UniqId:        uniqId,
			XForwardedFor: xForwardedFor,
			XRealIp:       xRealIp,
		}
		onOpenResp = callback.OnOpen(onOpenReq)
		if onOpenResp == nil {
			_ = wsConn.Close()
			return
		}
		//不允许连接
		if onOpenResp.Allow == false {
			if len(onOpenResp.Data) > 0 {
				timer.Timer.AfterFunc(time.Millisecond*100, func() {
					_ = wsConn.Close()
				})
			} else {
				_ = wsConn.Close()
			}
			return
		}
		//先将数据发给客户端，如果发送失败，则关闭连接，否则继续
		if len(onOpenResp.Data) > 0 {
			if configs.Config.Customer.SendMessageType == websocket.TextMessage {
				//utf8文本，直接写入
				err = wsConn.WriteMessage(configs.Config.Customer.SendMessageType, utils.StrToReadOnlyBytes(onOpenResp.Data))
			} else {
				//二进制文本，需要base64解码
				data := make([]byte, base64.StdEncoding.DecodedLen(len(onOpenResp.Data)))
				if _, err = base64.StdEncoding.Decode(data, utils.StrToReadOnlyBytes(onOpenResp.Data)); err != nil {
					err = wsConn.WriteMessage(configs.Config.Customer.SendMessageType, data)
				}
			}
			if err != nil {
				_ = wsConn.Close()
				return
			}
		}
	}

	//统计客户连接的打开次数
	metrics.Registry[metrics.ItemCustomerConnOpenCount].Meter.Mark(1)
	//添加到管理器中
	session := info.NewInfo(uniqId)
	wsConn.SetSession(session)
	manager.Manager.Set(uniqId, wsConn)
	//构建回调返回的关系
	if onOpenResp != nil {
		if onOpenResp.NewSession != "" {
			session.SetSession(onOpenResp.NewSession)
		}
		if onOpenResp.NewCustomerId != "" {
			session.SetCustomerId(onOpenResp.NewCustomerId)
			binder.Binder.Set(uniqId, onOpenResp.NewCustomerId)
		}
		if len(onOpenResp.NewTopics) > 0 {
			session.SubscribeTopics(onOpenResp.NewTopics)
			topic.Topic.SetBySlice(onOpenResp.NewTopics, uniqId)
		}
	}
	//需要将客户端连接打开的信息转发给business进程
	if worker := workerManager.Manager.Get(netsvrProtocol.Event_OnOpen); worker != nil {
		co := objPool.ConnOpen.Get()
		co.UniqId = uniqId
		co.RawQuery = r.URL.RawQuery
		co.SubProtocol = subProtocols
		co.XForwardedFor = xForwardedFor
		co.XRealIp = xRealIp
		co.RemoteAddr = remoteAddr
		if sendSize := worker.Send(co, netsvrProtocol.Cmd_ConnOpen); sendSize > 0 {
			//统计转发到business的次数与字节数
			metrics.Registry[metrics.ItemCustomerTransferCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(sendSize))
		}
		objPool.ConnOpen.Put(co)
	}
}

func onClose(conn *websocket.Conn, _ error) {
	//分配了session才算是真正打开过的连接
	session, ok := conn.SessionWithLock().(*info.Info)
	if !ok {
		return
	}
	//解除连接中持有的session
	conn.SetSession(nil)
	//开始执行关闭逻辑
	session.Lock()
	if session.GetUniqId() == "" {
		session.UnLock()
		return
	}
	//清空session
	uniqId, customerId, customerSession, topics := session.Clear()
	//释放锁
	session.UnLock()
	//统计客户连接的关闭次数
	metrics.Registry[metrics.ItemCustomerConnCloseCount].Meter.Mark(1)
	//从连接管理器中删除
	manager.Manager.Del(uniqId)
	//解除uniqId与customerId的关系
	if customerId != "" {
		//如果客户id为空，则没必要去解除绑定关系，因为解除绑定关系需要获取互斥锁
		binder.Binder.DelUniqId(uniqId)
	}
	//删除订阅关系
	topic.Topic.DelBySlice(topics, uniqId)
	if configs.Config.Customer.OnCloseCallbackApi != "" {
		onCloseReq := &callback.OnCloseReq{
			CustomerId: customerId,
			Session:    customerSession,
			Topics:     topics,
			UniqId:     uniqId,
		}
		//如果网关程序仅作推送消息的场景，则business进程是不存在的，所以这里需要回调，方便此场景下处理连接关闭的事件
		callback.OnClose(onCloseReq)
	}
	//将连接关闭的消息转发给business进程
	worker := workerManager.Manager.Get(netsvrProtocol.Event_OnClose)
	if worker != nil {
		cl := objPool.ConnClose.Get()
		cl.UniqId = uniqId
		cl.CustomerId = customerId
		cl.Session = customerSession
		cl.Topics = topics
		if sendSize := worker.Send(cl, netsvrProtocol.Cmd_ConnClose); sendSize > 0 {
			//统计客户数据转发到worker的次数与字节数情况
			metrics.Registry[metrics.ItemCustomerTransferCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(sendSize))
		}
		objPool.ConnClose.Put(cl)
	}
}

func onMessage(conn *websocket.Conn, _ websocket.MessageType, data []byte) {
	//检查是否为心跳消息
	if bytes.Equal(data, configs.Config.Customer.HeartbeatMessage) {
		//统计客户连接的心跳次数
		metrics.Registry[metrics.ItemCustomerHeartbeatCount].Meter.Mark(1)
		return
	}
	//从连接中拿出session
	session, ok := conn.SessionWithLock().(*info.Info)
	if !ok {
		return
	}
	//获取session中的数据
	session.Lock()
	allow := session.Allow()
	uniqId := session.GetUniqId()
	customerSession := session.GetSession()
	customerId := session.GetCustomerId()
	topics := session.GetTopics()
	session.UnLock()
	//限制数据包大小，溢出限制大小，直接丢弃该数据
	if len(data) > configs.Config.Customer.ReceivePackLimit {
		//关闭发生异常数据包的连接
		_ = conn.Close()
		//打日志，方便客户端排查问题
		log.Logger.Info().
			Str("uniqId", uniqId).
			Str("customerId", customerId).
			Str("customerSession", customerSession).
			Str("remoteAddr", conn.RemoteAddr().String()).
			Str("customerListenAddress", configs.Config.Customer.ListenAddress).
			Msg(MessageTooLarge)
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
			Str("remoteAddr", conn.RemoteAddr().String()).
			Str("customerListenAddress", configs.Config.Customer.ListenAddress).
			Msg(ConnectionMessageRateLimit)
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
			Str("remoteAddr", conn.RemoteAddr().String()).
			Str("customerListenAddress", configs.Config.Customer.ListenAddress).
			Msg(MessageRateLimit)
		return
	}
	//记录所有请求日志
	if configs.Config.LogLevel == "debug" {
		formatCustomerData(data, log.Logger.Debug()).
			Str("uniqId", uniqId).
			Str("customerId", customerId).
			Str("customerSession", customerSession).
			Str("remoteAddr", conn.RemoteAddr().String()).
			Str("customerListenAddress", configs.Config.Customer.ListenAddress).
			Send()
	}
	//获取能处理消息的business
	if worker := workerManager.Manager.Get(netsvrProtocol.Event_OnMessage); worker != nil {
		//编码数据成business需要的格式
		tf := objPool.Transfer.Get()
		tf.UniqId = uniqId
		tf.CustomerId = customerId
		tf.Session = customerSession
		tf.Topics = topics
		tf.Data = data
		//转发数据到business
		if sendSize := worker.Send(tf, netsvrProtocol.Cmd_Transfer); sendSize > 0 {
			//统计转发到business的次数与字节数
			metrics.Registry[metrics.ItemCustomerTransferCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(sendSize))
		}
		objPool.Transfer.Put(tf)
	}
}

func formatCustomerData(customerData []byte, event *zerolog.Event) *zerolog.Event {
	if configs.Config.Customer.SendMessageType == websocket.TextMessage {
		return event.Str("customerData", string(customerData))
	}
	return event.Hex("customerDataHex", customerData)
}

// pong客户端心跳帧
func pingMessageHandler(conn *websocket.Conn, _ string) {
	err := conn.SetWriteDeadline(time.Now().Add(configs.Config.Customer.SendDeadline))
	if err != nil {
		metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(configs.Config.Customer.HeartbeatMessage)))
		_ = conn.Close()
		return
	}
	err = conn.WriteMessage(websocket.PongMessage, configs.Config.Customer.HeartbeatMessage)
	if err != nil {
		metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(configs.Config.Customer.HeartbeatMessage)))
		_ = conn.Close()
		return
	}
	metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(configs.Config.Customer.HeartbeatMessage)))
	metrics.Registry[metrics.ItemCustomerHeartbeatCount].Meter.Mark(1)
}

func checkOrigin(r *http.Request) bool {
	if len(configs.Config.Customer.AllowOrigin) == 0 {
		return true
	}
	origin := r.Header.Get("Origin")
	for _, v := range configs.Config.Customer.AllowOrigin {
		if strings.Contains(origin, v) {
			return true
		}
	}
	return false
}
