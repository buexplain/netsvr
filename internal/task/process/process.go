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

package process

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/configs"
	"netsvr/internal/log"
	"time"
)

type GetCallback func(data []byte, taskConn net.Conn)
type PostCallback func(data []byte)

var getCallback map[netsvrProtocol.Cmd]GetCallback
var postCallback map[netsvrProtocol.Cmd]PostCallback

func init() {
	getCallback = map[netsvrProtocol.Cmd]GetCallback{
		netsvrProtocol.Cmd_UniqIdList:                   uniqIdList,
		netsvrProtocol.Cmd_UniqIdCount:                  uniqIdCount,
		netsvrProtocol.Cmd_TopicUniqIdList:              topicUniqIdList,
		netsvrProtocol.Cmd_TopicUniqIdCount:             topicUniqIdCount,
		netsvrProtocol.Cmd_TopicCount:                   topicCount,
		netsvrProtocol.Cmd_TopicList:                    topicList,
		netsvrProtocol.Cmd_ConnInfo:                     connInfo,
		netsvrProtocol.Cmd_ConnInfoByCustomerId:         connInfoByCustomerId,
		netsvrProtocol.Cmd_Metrics:                      metrics,
		netsvrProtocol.Cmd_CheckOnline:                  checkOnline,
		netsvrProtocol.Cmd_Limit:                        limit,
		netsvrProtocol.Cmd_CustomerIdCount:              customerIdCount,
		netsvrProtocol.Cmd_CustomerIdList:               customerIdList,
		netsvrProtocol.Cmd_TopicCustomerIdList:          topicCustomerIdList,
		netsvrProtocol.Cmd_TopicCustomerIdCount:         topicCustomerIdCount,
		netsvrProtocol.Cmd_TopicCustomerIdToUniqIdsList: topicCustomerIdToUniqIdsList,
	}
	postCallback = map[netsvrProtocol.Cmd]PostCallback{
		netsvrProtocol.Cmd_Broadcast:                  broadcast,
		netsvrProtocol.Cmd_ConnInfoUpdate:             connInfoUpdate,
		netsvrProtocol.Cmd_ConnInfoDelete:             connInfoDelete,
		netsvrProtocol.Cmd_ForceOffline:               forceOffline,
		netsvrProtocol.Cmd_ForceOfflineByCustomerId:   forceOfflineByCustomerId,
		netsvrProtocol.Cmd_ForceOfflineGuest:          forceOfflineGuest,
		netsvrProtocol.Cmd_Multicast:                  multicast,
		netsvrProtocol.Cmd_MulticastByCustomerId:      multicastByCustomerId,
		netsvrProtocol.Cmd_TopicPublish:               topicPublish,
		netsvrProtocol.Cmd_TopicPublishBulk:           topicPublishBulk,
		netsvrProtocol.Cmd_SingleCast:                 singleCast,
		netsvrProtocol.Cmd_SingleCastByCustomerId:     singleCastByCustomerId,
		netsvrProtocol.Cmd_SingleCastBulk:             singleCastBulk,
		netsvrProtocol.Cmd_SingleCastBulkByCustomerId: singleCastBulkByCustomerId,
		netsvrProtocol.Cmd_TopicSubscribe:             topicSubscribe,
		netsvrProtocol.Cmd_TopicUnsubscribe:           topicUnsubscribe,
		netsvrProtocol.Cmd_TopicDelete:                topicDelete,
	}
}

func Process(conn net.Conn) {
	defer func() {
		_ = conn.Close()
		if err := recover(); err != nil {
			log.Logger.Error().
				Stack().Err(nil).
				Type("recoverType", err).
				Interface("recover", err).
				Msg("Task coroutine is closed")
		} else {
			log.Logger.Debug().
				Msg("Task coroutine is closed")
		}
	}()
	dataLenBuf := make([]byte, 4)
	connReader := bufio.NewReaderSize(conn, configs.Config.Task.ReadBufferSize)
	for {
		//设置读超时时间，在这个时间之内，business端没有发数据过来，则会发生超时错误，导致连接被关闭
		if err := conn.SetReadDeadline(time.Now().Add(configs.Config.Task.ReadDeadline)); err != nil {
			return
		}
		//获取前4个字节，确定数据包长度
		dataLenBuf = dataLenBuf[:4]
		if _, err := io.ReadAtLeast(connReader, dataLenBuf, 4); err != nil {
			//读失败了，直接干掉这个连接，让business端重新连接进来，因为缓冲区的tcp流已经脏了，程序无法拆包
			//关掉重来，是最好的办法
			if err != io.EOF {
				log.Logger.Error().Err(err).Msg("Task read data length failed")
			}
			return
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		//发送是数据包太大，直接关闭business，如果dataLen非常地大，则有可能导致内存分配失败，从而导致整个进程崩溃
		if dataLen > configs.Config.Task.ReceivePackLimit {
			log.Logger.Error().
				Uint32("dataLen", dataLen).
				Uint32("receivePackLimit", configs.Config.Task.ReceivePackLimit).
				Msg("Task receive pack size overflow")
			return
		}
		//获取数据包，这里不必设置读取超时，因为接下来大大概率是有数据的，除非business不按包头包体的协议格式发送
		data := make([]byte, dataLen)
		if _, err := io.ReadAtLeast(connReader, data, len(data)); err != nil {
			if err != io.EOF {
				log.Logger.Error().Err(err).Msg("Task receive body failed")
			}
			return
		}
		//business发来心跳
		if bytes.Equal(configs.Config.Task.HeartbeatMessage, data) {
			continue
		}
		//处理业务
		if len(data) < 4 {
			log.Logger.Error().
				Hex("dataHex", data).
				Msg("Task unknown netsvrProtocol.Cmd")
			return
		}
		currentCmd := netsvrProtocol.Cmd(binary.BigEndian.Uint32(data[0:4]))
		if pCallback, ok := postCallback[currentCmd]; ok {
			fn := func() {
				defer func() {
					if err := recover(); err != nil {
						log.Logger.Error().
							Stack().Err(nil).
							Type("recoverType", err).
							Interface("recover", err).
							Str("cmd", currentCmd.String()).
							Msg("Task exec cmd failed")
					}
				}()
				pCallback(data[4:])
			}
			if err := goroutine.DefaultWorkerPool.Submit(fn); err != nil {
				//提交异步任务失败，则使用goroutine执行
				go fn()
				log.Logger.Warn().Err(err).
					Str("cmd", currentCmd.String()).
					Msg("Task submit to worker pool failed")
			}
		} else if gCallback, ok := getCallback[currentCmd]; ok {
			gCallback(data[4:], conn)
		} else {
			log.Logger.Error().
				Str("cmd", currentCmd.String()).
				Hex("dataHex", data).
				Msg("Task unknown netsvrProtocol.Cmd")
			return
		}
	}
}

func send(taskConn net.Conn, message proto.Message, cmd netsvrProtocol.Cmd) int {
	data := make([]byte, 8)
	//填充 cmd 字段 (大端序)
	binary.BigEndian.PutUint32(data[4:8], uint32(cmd))
	// 编码业务数据（如果有）
	if message != nil {
		var err error
		pm := proto.MarshalOptions{}
		data, err = pm.MarshalAppend(data, message)
		if err != nil {
			return 0
		}
	}
	//回填包长度字段（长度 = 包体字节数，不包含长度字段自身4字节）
	// message==nil 时: len=8, 8-4=4
	// message!=nil 时: len=8+body, (8+body)-4 = 4+body
	binary.BigEndian.PutUint32(data[0:4], uint32(len(data)-4))
	//发送出去
	var err error
	var writeLen int
	dataRef := data
	//写入到连接中
	for {
		//设置写超时
		if err = taskConn.SetWriteDeadline(time.Now().Add(configs.Config.Task.SendDeadline)); err != nil {
			_ = taskConn.Close()
			log.Logger.Error().
				Uint32("cmd", uint32(cmd)).
				Msg("Task SetWriteDeadline failed")
			return 0
		}
		//写入数据
		writeLen, err = taskConn.Write(data)
		//写入成功
		if err == nil {
			//完全写入成功
			if writeLen == len(data) {
				return len(dataRef)
			}
			//短写：继续发送剩余字节
			data = data[writeLen:]
			time.Sleep(time.Millisecond * 10)
			continue
		}
		//判断错误类型：超时且未污染连接时可安全丢弃
		var opErr *net.OpError
		if errors.As(err, &opErr) && opErr.Timeout() && len(dataRef) == len(data[writeLen:]) {
			log.Logger.Error().Str("cmd", cmd.String()).Err(err).
				Msg("Task send failed and discard message")
			return 0
		}
		//其他错误或已写入部分数据：连接状态不可恢复，强制关闭
		_ = taskConn.Close()
		log.Logger.Error().Str("cmd", cmd.String()).Err(err).
			Msg("Task send failed and force close conn")
		return 0
	}
}
