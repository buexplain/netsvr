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
	"compress/flate"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
	"io"
	"netsvr/configs"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/timer"
	"netsvr/internal/utils/buffer"
	"netsvr/internal/wsServer"
	"time"
)

var flateHelper wsflate.Helper

var compressingValue = 256

func init() {
	if configs.Config.Autobahn {
		compressingValue = 0
	}
	flateHelper = wsflate.Helper{
		Compressor: func(w io.Writer) wsflate.Compressor {
			// No error can be returned here as NewWriter() doc says.
			f, _ := flate.NewWriter(w, configs.Config.Customer.CompressionLevel)
			return f
		},
		Decompressor: func(r io.Reader) wsflate.Decompressor {
			return flate.NewReader(r)
		},
	}
}

type Message struct {
	messageType  ws.OpCode
	data         []byte
	compressed   []byte
	uncompressed []byte
}

func (r *Message) WriteTo(conn gnet.Conn) bool {
	wsCodec, ok := conn.Context().(*wsServer.Codec)
	if !ok || wsCodec.IsClosed() {
		return false
	}
	//需要压缩，并且数据值得压缩
	if wsCodec.IsCompress() && len(r.data) > compressingValue {
		//先压缩数据
		if r.compressed == nil {
			compressBytes := byteslice.Get(len(r.data)) //申请一块用于压缩的内存
			defer byteslice.Put(compressBytes)          //回收内存
			buff := buffer.New(compressBytes[:0])       //创建一个buffer容器
			defer buff.Discard()                        //解除底层的内存引用
			err := flateHelper.CompressTo(buff, r.data)
			if err != nil {
				log.Logger.Error().Err(err).Msg("compress websocket message error")
				metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(r.data)))
				return false
			}
			frame := ws.NewFrame(r.messageType, true, buff.Bytes())
			frame.Header.Rsv = ws.Rsv(true, false, false)
			payloadLen := len(frame.Payload)
			headerSize := 2
			if payloadLen >= 126 && payloadLen < 65536 {
				headerSize += 2
			} else if payloadLen >= 65536 {
				headerSize += 8
			}
			buff.Set(make([]byte, 0, headerSize+payloadLen)) //复用buffer容器
			err = ws.WriteFrame(buff, frame)
			if err != nil {
				metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(payloadLen))
				return false
			}
			r.compressed = buff.Bytes()
		}
		//再发送数据
		err := conn.AsyncWrite(r.compressed, nil)
		if err == nil {
			metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(r.compressed)))
			return true
		}
		//发送失败，则关闭连接
		log.Logger.Error().Err(err).Msg("write message to websocket conn error")
		metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(r.compressed)))
		return false
	}
	//不压缩，先判断是否已经编码成frame
	if r.uncompressed == nil {
		payloadLen := len(r.data)
		headerSize := 2
		if payloadLen >= 126 && payloadLen < 65536 {
			headerSize += 2
		} else if payloadLen >= 65536 {
			headerSize += 8
		}
		buff := buffer.New(make([]byte, 0, headerSize+payloadLen)) //创建一个buffer容器
		defer buff.Discard()                                       //解除底层的内存引用
		frame := ws.NewFrame(r.messageType, true, r.data)
		err := ws.WriteFrame(buff, frame)
		if err != nil {
			metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(payloadLen))
			return false
		}
		r.uncompressed = buff.Bytes()
	}
	//frame已经编码ok，发送数据
	err := conn.AsyncWrite(r.uncompressed, nil)
	if err == nil {
		metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(r.uncompressed)))
		return true
	}
	//发送失败，则关闭连接
	log.Logger.Error().Err(err).Msg("write message to websocket conn error")
	metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(r.uncompressed)))
	return false
}

func NewMessage(messageType ws.OpCode, data []byte) *Message {
	return &Message{
		messageType: messageType,
		data:        data,
	}
}

// WriteMessage 发送数据
func WriteMessage(conn gnet.Conn, messageType ws.OpCode, data []byte) bool {
	wsCodec, ok := conn.Context().(*wsServer.Codec)
	if !ok || wsCodec.IsClosed() {
		return false
	}
	//需要压缩，并且数据值得压缩
	if wsCodec.IsCompress() && len(data) > compressingValue {
		compressBytes := byteslice.Get(len(data)) //申请一块用于压缩的内存
		defer byteslice.Put(compressBytes)        //回收内存
		buff := buffer.New(compressBytes[:0])     //创建一个buffer容器
		defer buff.Discard()                      //解除底层的内存引用
		err := flateHelper.CompressTo(buff, data)
		if err != nil {
			log.Logger.Error().Err(err).Msg("compress websocket message error")
			metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(data)))
			return false
		}
		frame := ws.NewFrame(messageType, true, buff.Bytes())
		frame.Header.Rsv = ws.Rsv(true, false, false)
		payloadLen := len(frame.Payload)
		headerSize := 2
		if payloadLen >= 126 && payloadLen < 65536 {
			headerSize += 2
		} else if payloadLen >= 65536 {
			headerSize += 8
		}
		buff.Set(make([]byte, 0, headerSize+payloadLen)) //复用buffer容器
		err = ws.WriteFrame(buff, frame)
		if err != nil {
			metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(payloadLen))
			return false
		}
		//再发送数据
		encodedBytes := buff.Bytes()
		err = conn.AsyncWrite(encodedBytes, nil)
		if err == nil {
			metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(encodedBytes)))
			return true
		}
		//发送失败，则关闭连接
		log.Logger.Error().Err(err).Msg("write message to websocket conn error")
		metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(encodedBytes)))
		return false
	}
	//不压缩
	payloadLen := len(data)
	headerSize := 2
	if payloadLen >= 126 && payloadLen < 65536 {
		headerSize += 2
	} else if payloadLen >= 65536 {
		headerSize += 8
	}
	buff := buffer.New(make([]byte, 0, headerSize+payloadLen)) //创建一个buffer容器
	defer buff.Discard()                                       //解除底层的内存引用
	frame := ws.NewFrame(messageType, true, data)
	err := ws.WriteFrame(buff, frame)
	if err != nil {
		metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(payloadLen))
		return false
	}
	//frame已经编码ok，发送数据
	encodedBytes := buff.Bytes()
	err = conn.AsyncWrite(encodedBytes, nil)
	if err == nil {
		metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(encodedBytes)))
		return true
	}
	//发送失败，则关闭连接
	log.Logger.Error().Err(err).Msg("write message to websocket conn error")
	metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(encodedBytes)))
	return false
}

// WriteClose 构建关闭帧，并发送关闭帧
func WriteClose(conn gnet.Conn, statusCode ws.StatusCode, err error) {
	WriteCloseFrame(conn, BuildCloseFrame(statusCode, err))
}

// WriteCloseFrame 发送关闭帧
func WriteCloseFrame(conn gnet.Conn, closeFrame []byte) {
	wsCodec, ok := conn.Context().(*wsServer.Codec)
	if !ok || wsCodec.IsClosed() {
		//连接已经关闭
		return
	}
	err := conn.AsyncWrite(closeFrame, nil)
	if err == nil {
		wsCodec.SetClosed()
		//5秒后强制关闭连接
		timer.Timer.AfterFunc(time.Second*5, func() {
			_ = conn.Close()
		})
	} else {
		//发送失败，则强制关闭连接
		_ = conn.Close()
	}
}

// BuildCloseFrame 构建关闭帧
func BuildCloseFrame(statusCode ws.StatusCode, err error) []byte {
	buff := bytes.NewBuffer(make([]byte, 0, 127))
	payload := ws.NewCloseFrameBody(
		statusCode,
		err.Error(),
	)
	frame := ws.NewFrame(ws.OpClose, true, payload)
	_ = ws.WriteFrame(buff, frame)
	return buff.Bytes()
}
