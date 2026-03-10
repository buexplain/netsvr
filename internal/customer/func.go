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
	"io"
	"netsvr/configs"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/timer"
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
			var err error
			var buff *bytes.Buffer
			var frame ws.Frame
			var data []byte
			data, err = flateHelper.Compress(r.data)
			if err != nil {
				log.Logger.Error().Err(err).Msg("compress websocket message error")
				metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(r.data)))
				return false
			}
			frame = ws.NewFrame(r.messageType, true, data)
			frame.Header.Rsv = ws.Rsv(true, false, false)
			buff = bytes.NewBuffer(make([]byte, 0, 4+len(data)))
			err = ws.WriteFrame(buff, frame)
			if err != nil {
				metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(r.data)))
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
		var err error
		var buff *bytes.Buffer
		var frame ws.Frame
		frame = ws.NewFrame(r.messageType, true, r.data)
		buff = bytes.NewBuffer(make([]byte, 0, 4+len(r.data)))
		err = ws.WriteFrame(buff, frame)
		if err != nil {
			metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(r.data)))
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
	var err error
	var buff *bytes.Buffer
	var frame ws.Frame
	//需要压缩，并且数据值得压缩
	if wsCodec.IsCompress() && len(data) > compressingValue {
		compressed, err := flateHelper.Compress(data)
		if err != nil {
			metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(data)))
			return false
		}
		frame = ws.NewFrame(messageType, true, compressed)
		frame.Header.Rsv = ws.Rsv(true, false, false)
	} else {
		//不压缩
		frame = ws.NewFrame(messageType, true, data)
	}
	//创建frame
	buff = bytes.NewBuffer(make([]byte, 0, 4+len(data)))
	err = ws.WriteFrame(buff, frame)
	if err != nil {
		metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(data)))
		return false
	}
	//发送数据
	err = conn.AsyncWrite(buff.Bytes(), nil)
	if err == nil {
		metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(data)))
		return true
	}
	//发送失败，则关闭连接
	log.Logger.Error().Err(err).Msg("write message to websocket conn error")
	metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(data)))
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
