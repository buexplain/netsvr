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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"github.com/lesismal/llib/std/crypto/tls"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"net/http"
	"netsvr/configs"
	"netsvr/internal/customer/info"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/token"
	"netsvr/internal/customer/topic"
	"netsvr/internal/limit"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/objPool"
	"netsvr/internal/utils"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/quit"
	"os"
	"strings"
	"time"
)

var server *nbhttp.Server

// ConnectRateLimit 不能轻易改变这些常量的值，因为websocket客户端代码极有可能判断这些字符串做出下一步处理，改变这些字符会导致这些客户端的代码失效
const ConnectRateLimit = "Connect rate limited"
const MessageRateLimit = "Message rate limited"
const ServiceShutdown = "Service shutdown"
const MessageTooLarge = "Message too large"
const WorkerIdWrong = "WorkerId wrong"
const CustomUniqIdWrong = "Custom uniqId wrong"
const ConnOpenWorkerNotFound = "Connection open worker not found"

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
		Network: "tcp",
		Name:    "customer",
		MaxLoad: configs.Config.Customer.MaxOnlineNum,
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
		//限流检查
		//之所以要判断workerId大于0，是因为有可能业务方并不关心连接的打开信息，连接打开信息不会传递到业务方，则不必限流
		if configs.Config.Customer.ConnOpenWorkerId > 0 && limit.Manager.Allow(configs.Config.Customer.ConnOpenWorkerId) == false {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write(utils.StrToReadOnlyBytes(ConnectRateLimit))
			log.Logger.Error().Int("workerId", configs.Config.Customer.ConnOpenWorkerId).Str("customerListenAddress", configs.Config.Customer.ListenAddress).Msg(ConnectRateLimit)
			return
		}
	}
	var uniqId string
	//获取uniqId
	if configs.Config.Customer.ConnOpenCustomUniqIdKey == "" {
		uniqId = utils.UniqId()
	} else {
		query := r.URL.Query()
		uniqId = query.Get(configs.Config.Customer.ConnOpenCustomUniqIdKey)
		if l := len(uniqId); l == 0 || l > 128 || !token.Token.Exist(query.Get("token")) {
			//如果客户端传递的uniqId太长，则会被拒绝，这么做的目的是避免太长的uniqId对网关的稳定性不利
			//校验连接的token是避免闲杂人等的接入
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(utils.StrToReadOnlyBytes(CustomUniqIdWrong))
			return
		}
	}
	upgrade := websocket.NewUpgrader()
	upgrade.KeepaliveTime = configs.Config.Customer.ReadDeadline
	upgrade.CheckOrigin = checkOrigin
	upgrade.SetPingHandler(pingMessageHandler)
	upgrade.SetPongHandler(pongMessageHandler)
	upgrade.OnOpen(nil)
	upgrade.OnClose(onClose)
	upgrade.OnMessage(onMessage)
	//处理websocket子协议
	upgrade.Subprotocols = nil
	subProtocols := utils.ParseSubProtocols(r)
	var responseHeader http.Header
	if subProtocols != nil {
		//返回任意一个子协议，保证连接升级成功
		responseHeader = http.Header{}
		responseHeader["Sec-Websocket-Protocol"] = subProtocols[0:1]
	}
	//开始升级
	wsConn, err := upgrade.Upgrade(w, r, responseHeader)
	if err != nil {
		log.Logger.Error().Err(err).Str("customerListenAddress", configs.Config.Customer.ListenAddress).Msg("Customer websocket upgrade failed")
		return
	}
	//升级成功
	//统计打开连接次数
	metrics.Registry[metrics.ItemCustomerConnOpen].Meter.Mark(1)
	//添加到管理器中
	session := info.NewInfo(uniqId)
	wsConn.SetSession(session)
	manager.Manager.Set(uniqId, wsConn)
	log.Logger.Debug().Str("uniqId", uniqId).Str("customerListenAddress", configs.Config.Customer.ListenAddress).Msg("Customer websocket open")
	//获取能够处理连接打开信息的business
	if configs.Config.Customer.ConnOpenWorkerId == 0 {
		//配置为0，表示当前业务不关心连接的打开信息
		return
	}
	worker := workerManager.Manager.Get(configs.Config.Customer.ConnOpenWorkerId)
	if worker == nil {
		//响应一个错误给到客户端
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(utils.StrToReadOnlyBytes(ConnOpenWorkerNotFound))
		//配置了处理的workerId，但是又没找到具体的业务进程，则打错误日志告警，因为此时有可能是business进程挂了
		log.Logger.Error().Int("workerId", configs.Config.Customer.ConnOpenWorkerId).Str("customerListenAddress", configs.Config.Customer.ListenAddress).Msg("Not found process conn open business")
		//关闭连接
		_ = wsConn.Close()
		return
	}
	//连接打开消息回传给business
	co := objPool.ConnOpen.Get()
	co.SubProtocol = subProtocols
	co.XForwardedFor = r.Header.Get("X-Forwarded-For")
	if co.XForwardedFor == "" {
		//没有代理信息，则将直接与网关连接的ip转发给business
		co.XForwardedFor = strings.Split(wsConn.RemoteAddr().String(), ":")[0]
	}
	co.RawQuery = r.URL.RawQuery
	co.UniqId = uniqId
	if sendSize := worker.Send(co, netsvrProtocol.Cmd_ConnOpen); sendSize > 0 {
		//统计转发到business的次数与字节数
		metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(sendSize + 4)) //加上4字节，是因为tcp包头的缘故
	}
	objPool.ConnOpen.Put(co)
}

// ping客户端心跳帧
func pongMessageHandler(_ *websocket.Conn, _ string) {
}

// pong客户端心跳帧
func pingMessageHandler(conn *websocket.Conn, _ string) {
	//响应客户端心跳
	if err := conn.WriteMessage(websocket.PongMessage, netsvrProtocol.PongMessage); err == nil {
		metrics.Registry[metrics.ItemCustomerHeartbeat].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteNumber].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(netsvrProtocol.PongMessage)))
	} else {
		_ = conn.Close()
	}
}

func onClose(conn *websocket.Conn, _ error) {
	//分配了session才算是真正打开过的连接
	session, ok := conn.Session().(*info.Info)
	if !ok {
		return
	}
	//统计关闭连接次数
	metrics.Registry[metrics.ItemCustomerConnClose].Meter.Mark(1)
	session.MuxLock()
	//关闭info
	session.Close()
	//清空info，并返回相关数据
	topics, uniqId, customerSession := session.Clear()
	if uniqId == "" {
		session.MuxUnLock()
		//当前连接已经被清空了uniqId，无需进行接下来的逻辑
		return
	}
	//从连接管理器中删除
	manager.Manager.Del(uniqId)
	//删除订阅关系
	topic.Topic.DelByMap(topics, uniqId, "")
	//释放锁
	session.MuxUnLock()
	//连接关闭消息回传给business
	if configs.Config.Customer.ConnCloseWorkerId == 0 {
		//配置为0，表示当前业务不关心连接的关闭信息
		return
	}
	worker := workerManager.Manager.Get(configs.Config.Customer.ConnCloseWorkerId)
	if worker == nil {
		log.Logger.Error().Int("workerId", configs.Config.Customer.ConnCloseWorkerId).Str("customerListenAddress", configs.Config.Customer.ListenAddress).Msg("Not found process conn close business")
		return
	}
	//转发数据到business
	cl := objPool.ConnClose.Get()
	cl.UniqId = uniqId
	cl.Session = customerSession
	if sendSize := worker.Send(cl, netsvrProtocol.Cmd_ConnClose); sendSize > 0 {
		//统计转发到business的次数与字节数
		metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(sendSize + 4)) //加上4字节，是因为tcp包头的缘故
	}
	objPool.ConnClose.Put(cl)
}

func onMessage(conn *websocket.Conn, messageType websocket.MessageType, data []byte) {
	//检查是否为心跳消息
	if bytes.Equal(data, netsvrProtocol.PingMessage) {
		//响应客户端心跳
		if err := conn.WriteMessage(messageType, netsvrProtocol.PongMessage); err == nil {
			metrics.Registry[metrics.ItemCustomerHeartbeat].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteNumber].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(netsvrProtocol.PongMessage)))
		} else {
			_ = conn.Close()
		}
		return
	}
	//限制数据包大小
	if len(data)-3 > configs.Config.Customer.ReceivePackLimit {
		if err := conn.WriteMessage(messageType, utils.StrToReadOnlyBytes(MessageTooLarge)); err == nil {
			metrics.Registry[metrics.ItemCustomerWriteNumber].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(MessageTooLarge)))
		} else {
			_ = conn.Close()
		}
		//溢出限制大小，直接丢弃该数据
		return
	}
	//读取前三个字节，转成business的workerId
	workerId := utils.BytesToInt(data, 3)
	//检查workerId是否合法
	if workerId < netsvrProtocol.WorkerIdMin || workerId > netsvrProtocol.WorkerIdMax {
		if err := conn.WriteMessage(messageType, utils.StrToReadOnlyBytes(WorkerIdWrong)); err == nil {
			metrics.Registry[metrics.ItemCustomerWriteNumber].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(WorkerIdWrong)))
		} else {
			_ = conn.Close()
		}
		return
	}
	//限流检查
	if limit.Manager.Allow(workerId) == false {
		//触发了限流也要打错误日志告警，因为这个时候有可能是因为客户消息太多了，网关部署地太少
		log.Logger.Error().Int("workerId", workerId).Str("remoteAddr", conn.RemoteAddr().String()).Str("customerListenAddress", configs.Config.Customer.ListenAddress).Msg(MessageRateLimit)
		return
	}
	//从连接中拿出session
	session, ok := conn.Session().(*info.Info)
	if !ok {
		return
	}
	//获取能处理消息的business
	worker := workerManager.Manager.Get(workerId)
	if worker == nil {
		//这里打error日志告警，假设客户端是传递了正确的workerId，但是服务端的业务进程可能挂了，导致没找到，这时候打告警日志是必须的
		//当然客户端有可能是恶意乱传，所以记录下对方的远程地址也是必须的
		log.Logger.Error().Int("workerId", workerId).Str("remoteAddr", conn.RemoteAddr().String()).Str("customerListenAddress", configs.Config.Customer.ListenAddress).Str("session", session.GetSession()).Bytes("customerToWorkerData", data[3:]).Msg("Not found business")
		return
	}
	//编码数据成business需要的格式
	tf := objPool.Transfer.Get()
	tf.Data = data[3:]
	session.GetToProtocolTransfer(tf)
	//转发数据到business
	if sendSize := worker.Send(tf, netsvrProtocol.Cmd_Transfer); sendSize > 0 {
		//统计转发到business的次数与字节数
		metrics.Registry[metrics.ItemCustomerTransferNumber].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerTransferByte].Meter.Mark(int64(sendSize + 4)) //加上4字节，是因为tcp包头的缘故
	}
	objPool.Transfer.Put(tf)
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
