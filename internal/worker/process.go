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
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/configs"
	"netsvr/internal/log"
	"time"
)

func process(workerConn *Conn) {
	defer func() {
		workerConn.Close()
		if err := recover(); err != nil {
			log.Logger.Error().
				Stack().
				Any("panic", err).
				Int32("events", workerConn.GetEvents()).
				Str("connId", workerConn.connId).
				Msg("Worker receive coroutine is closed")
			_ = workerConn.conn.Close()
		} else {
			log.Logger.Debug().
				Int32("events", workerConn.GetEvents()).
				Str("connId", workerConn.connId).
				Msg("Worker receive coroutine is closed")
			_ = workerConn.conn.Close()
		}
	}()

	// 禁用 Nagle 算法，减少小包延迟
	if tcpConn, ok := workerConn.conn.(*net.TCPConn); ok {
		if err := tcpConn.SetNoDelay(true); err != nil {
			log.Logger.Warn().Err(err).
				Int32("events", workerConn.GetEvents()).
				Str("connId", workerConn.connId).
				Msg("Worker SetNoDelay failed")
		}
	}

	dataLenBuf := make([]byte, 4)
	connReader := bufio.NewReaderSize(workerConn.conn, configs.Config.Worker.ReadBufferSize)
	for {
		//设置读超时时间，在这个时间之内，business端没有发数据过来，则会发生超时错误，导致连接被关闭
		if err := workerConn.conn.SetReadDeadline(time.Now().Add(configs.Config.Worker.ReadDeadline)); err != nil {
			return
		}
		//获取前4个字节，确定数据包长度
		dataLenBuf = dataLenBuf[:4]
		if _, err := io.ReadAtLeast(connReader, dataLenBuf, 4); err != nil {
			//读失败了，直接干掉这个连接，让business端重新连接进来，因为缓冲区的tcp流已经脏了，程序无法拆包
			//关掉重来，是最好的办法
			if err != io.EOF {
				log.Logger.Error().
					Int32("events", workerConn.GetEvents()).
					Str("connId", workerConn.connId).
					Err(err).Msg("Worker receive data length failed")
			}
			return
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		//发送是数据包太大，直接关闭business，如果dataLen非常地大，则有可能导致内存分配失败，从而导致整个进程崩溃
		if dataLen > configs.Config.Worker.ReceivePackLimit {
			log.Logger.Error().
				Int32("events", workerConn.GetEvents()).
				Str("connId", workerConn.connId).
				Uint32("dataLen", dataLen).
				Uint32("receivePackLimit", configs.Config.Worker.ReceivePackLimit).
				Msg("Worker receive pack size overflow")
			return
		}
		//获取数据包，这里不必设置读取超时，因为接下来大大概率是有数据的，除非business不按包头包体的协议格式发送
		data := make([]byte, dataLen)
		if _, err := io.ReadAtLeast(connReader, data, len(data)); err != nil {
			if err != io.EOF {
				log.Logger.Error().
					Int32("events", workerConn.GetEvents()).
					Str("connId", workerConn.connId).
					Err(err).Msg("Worker receive body failed")
			}
			return
		}
		//business发来心跳
		if bytes.Equal(configs.Config.Worker.HeartbeatMessage, data) {
			continue
		}
		//处理业务
		if len(data) < 4 {
			log.Logger.Error().
				Int32("events", workerConn.GetEvents()).
				Str("connId", workerConn.connId).
				Hex("dataHex", data).
				Msg("Worker unknown netsvrProtocol.Cmd")
			return
		}
		currentCmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(data[0:4]))
		if currentCmd == netsvrProtocol.Cmd_Register {
			func() {
				defer func() {
					if err := recover(); err != nil {
						log.Logger.Error().
							Stack().
							Any("panic", err).
							Str("cmd", currentCmd.String()).
							Msg("Worker exec cmd failed")
					}
				}()
				register(data[4:], workerConn)
			}()
		} else if currentCmd == netsvrProtocol.Cmd_Unregister {
			func() {
				defer func() {
					if err := recover(); err != nil {
						log.Logger.Error().
							Stack().
							Any("panic", err).
							Str("cmd", currentCmd.String()).
							Msg("Worker exec cmd failed")
					}
				}()
				unregister(data[4:], workerConn)
			}()
		} else {
			log.Logger.Error().
				Int32("events", workerConn.GetEvents()).
				Str("connId", workerConn.connId).
				Str("cmd", currentCmd.String()).
				Hex("dataHex", data).
				Msg("Worker unknown netsvrProtocol.Cmd")
			return
		}
	}
}

// unregister business取消注册状态
func unregister(param []byte, workerConn *Conn) {
	payload := netsvrProtocol.UnRegisterReq{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		log.Logger.Error().
			Int32("events", workerConn.GetEvents()).
			Str("connId", workerConn.connId).
			Err(err).Msg("Proto unmarshal netsvrProtocol.UnRegisterReq failed")
		return
	}
	if Manager.Del(payload.ConnId) {
		log.Logger.Info().
			Int32("events", workerConn.GetEvents()).
			Str("connId", workerConn.connId).
			Str("remoteAddr", workerConn.GetConnRemoteAddr()).Msg("Unregister a business")
	}
	//必须回复给business，否则business端会认为没有收到数据，从而一直等待
	workerConn.Send(&netsvrProtocol.UnRegisterResp{}, netsvrProtocol.Cmd_Unregister)
}

// register 注册business进程
func register(param []byte, workerConn *Conn) {
	payload := netsvrProtocol.RegisterReq{}
	ret := &netsvrProtocol.RegisterResp{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		ret.Code = netsvrProtocol.RegisterRespCode_UnmarshalError
		log.Logger.Error().
			Int32("events", workerConn.GetEvents()).
			Str("connId", workerConn.connId).
			Err(err).Msg("Proto unmarshal netsvrProtocol.RegisterReq failed")
		ret.Message = ret.Code.String()
		workerConn.Send(ret, netsvrProtocol.Cmd_Register)
		return
	}
	//判断设置的事件是否有效
	invalidEvent := true
	for _, v := range netsvrProtocol.Event_value {
		if v == int32(netsvrProtocol.Event_Placeholder) {
			continue
		}
		if payload.Events&v == v {
			invalidEvent = false
			break
		}
	}
	if invalidEvent {
		ret.Code = netsvrProtocol.RegisterRespCode_InvalidEvent
		log.Logger.Error().
			Str("connId", workerConn.connId).
			Int32("events", payload.Events).Msg("Invalid RegisterReqEvent")
		ret.Message = ret.Code.String()
		workerConn.Send(ret, netsvrProtocol.Cmd_Register)
		return
	}
	//检查当前的business连接是否已经注册，不允许重复注册
	if workerConn.GetEvents() > 0 {
		ret.Code = netsvrProtocol.RegisterRespCode_DuplicateRegister
		log.Logger.Error().
			Int32("events", workerConn.GetEvents()).
			Str("connId", workerConn.connId).
			Msg("Duplicate register are not allowed")
		ret.Message = ret.Code.String()
		workerConn.Send(ret, netsvrProtocol.Cmd_Register)
		return
	}
	//设置business连接的events
	workerConn.SetEvents(payload.Events)
	//先将注册结果返回给business
	ret.Code = netsvrProtocol.RegisterRespCode_Success
	ret.Message = ret.Code.String()
	ret.ConnId = workerConn.GetConnId()
	workerConn.Send(ret, netsvrProtocol.Cmd_Register)
	//记录日志
	log.Logger.Info().
		Str("remoteAddr", workerConn.GetConnRemoteAddr()).
		Int32("events", payload.Events).
		Str("connId", workerConn.GetConnId()).
		Msg("Register a business")
	//最后，将该business连接登记到worker管理器中，一定要最后加入，确保注册结果的数据是第一个到达business进程，因为加入管理器后随时可能被转发数据
	Manager.Set(workerConn)
}
