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
	"compress/flate"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
	"io"
	"netsvr/configs"
	"netsvr/internal/log"
	"netsvr/internal/metrics"
	"netsvr/internal/utils/buffer"
	"netsvr/internal/wsServer"
	"sync"
)

// flateHelper 供 wsflate 对单条消息做 deflate 压缩/解压，与 gobwas 的 permessage-deflate 扩展配套使用。
var flateHelper wsflate.Helper

// compressingValue 开启压缩的连接上，payload 超过该字节数才走压缩路径；Autobahn 测试下设为 0 表示始终尝试压缩。
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

// Frame 表示一条待下行写入的 WebSocket 业务消息：负责按连接是否开启压缩，把 data 编成一条完整帧字节流，并通过 gnet 异步写出。
//
// 使用约定（与 FrameObjPool 配合）：
//   - 通过 FrameObjPool.Get 取得实例，用完后必须 FrameObjPool.Put；Put 前会 reset，避免把编码缓存泄漏到池里。
//   - data 仅保存切片头，调用方不得在异步写完成前改写该切片底层内容（与 proto 的 []byte 字段常见用法一致）。
//   - 可对多个 conn 连续调用 WriteTo：第一次命中路径时完成压缩/组帧并缓存到 compressed 或 uncompressed，后续连接复用同一份线字节，避免重复压缩与 WriteFrame。
//   - AsyncWriteOnSafe 只入队写任务；返回后 gnet 可能尚未真正写 socket。线字节由 gnet 出站缓冲拷贝或短时引用，Frame 结构体上的字段可被 reset，与队列中已捕获的 []byte 不冲突。
type Frame struct {
	messageType ws.OpCode // 帧操作码（如 Text / Binary）
	data        []byte    // 业务 payload，来自上游切片引用
	// 已协商 permessage-deflate 且 payload 超过阈值时，WriteFrame 后的完整线字节（含 WS 头），供多连接复用
	compressed []byte
	// 未压缩或本连接不压缩时，WriteFrame 后的完整线字节（含 WS 头），供多连接复用
	uncompressed []byte
}

func (fr *Frame) set(messageType ws.OpCode, data []byte) {
	fr.messageType = messageType
	fr.data = data
}

// reset 归还池前清空引用，防止压缩/未压缩缓存与下一条消息的 data 串用。
func (fr *Frame) reset() {
	fr.data = nil
	fr.compressed = nil
	fr.uncompressed = nil
}

// WriteTo 将当前 data 编码为一条 WebSocket 数据帧并异步写入 conn。成功返回 true；连接已关或编码/入队失败返回 false。
// 同一 Frame 对多个 conn 调用时，按各 conn 的 IsCompressOnSafe 与阈值选择分支；已缓存的 compressed/uncompressed 会直接复用。
func (fr *Frame) WriteTo(conn *wsServer.Conn) bool {
	if conn.IsClosedOnSafe() {
		return false
	}
	//需要压缩，并且数据值得压缩
	if conn.IsCompressOnSafe() && len(fr.data) > compressingValue {
		//先压缩数据
		if fr.compressed == nil {
			compressBytes := byteslice.Get(len(fr.data)) //申请一块用于压缩的内存
			defer byteslice.Put(compressBytes)           //回收内存
			buff := buffer.New(compressBytes[:0])        //创建一个buffer容器
			defer buff.Discard()                         //解除底层的内存引用
			err := flateHelper.CompressTo(buff, fr.data)
			if err != nil {
				log.Logger.Error().Err(err).Msg("compress websocket message error")
				metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(fr.data)))
				return false
			}
			frame := ws.NewFrame(fr.messageType, true, buff.Bytes())
			frame.Header.Rsv = ws.Rsv(true, false, false)
			payloadLen := len(frame.Payload)
			headerSize := 2
			if payloadLen >= 126 && payloadLen < 65536 {
				headerSize += 2
			} else if payloadLen >= 65536 {
				headerSize += 8
			}
			// 换一块新 backing 写 WS 头+体；fr.compressed 指向该块而非 compressBytes，故 defer byteslice.Put(compressBytes) 安全
			buff.Set(make([]byte, 0, headerSize+payloadLen)) //复用buffer容器
			err = ws.WriteFrame(buff, frame)
			if err != nil {
				metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
				metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(payloadLen))
				return false
			}
			fr.compressed = buff.Bytes()
		}
		//再发送数据
		err := conn.AsyncWriteOnSafe(fr.compressed)
		if err == nil {
			metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(fr.compressed)))
			return true
		}
		//发送失败，则关闭连接
		log.Logger.Error().Err(err).Msg("write message to websocket conn error")
		metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(fr.compressed)))
		return false
	}
	//不压缩，先判断是否已经编码成frame
	if fr.uncompressed == nil {
		payloadLen := len(fr.data)
		headerSize := 2
		if payloadLen >= 126 && payloadLen < 65536 {
			headerSize += 2
		} else if payloadLen >= 65536 {
			headerSize += 8
		}
		buff := buffer.New(make([]byte, 0, headerSize+payloadLen)) //创建一个buffer容器
		defer buff.Discard()                                       //解除底层的内存引用
		frame := ws.NewFrame(fr.messageType, true, fr.data)
		err := ws.WriteFrame(buff, frame)
		if err != nil {
			metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
			metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(payloadLen))
			return false
		}
		fr.uncompressed = buff.Bytes()
	}
	//frame已经编码ok，发送数据
	err := conn.AsyncWriteOnSafe(fr.uncompressed)
	if err == nil {
		metrics.Registry[metrics.ItemCustomerWriteCount].Meter.Mark(1)
		metrics.Registry[metrics.ItemCustomerWriteByte].Meter.Mark(int64(len(fr.uncompressed)))
		return true
	}
	//发送失败，则关闭连接
	log.Logger.Error().Err(err).Msg("write message to websocket conn error")
	metrics.Registry[metrics.ItemCustomerWriteFailedCount].Meter.Mark(1)
	metrics.Registry[metrics.ItemCustomerWriteFailedByte].Meter.Mark(int64(len(fr.uncompressed)))
	return false
}

// framePool 下行 Frame 的 sync.Pool 封装，降低高频单播/广播场景下的堆分配。
type framePool struct {
	pool sync.Pool
}

// FrameObjPool 全局 Frame 池。Get/Put 须成对；若需对多个连接写同一条内容，可在一次 Get 与一次 Put 之间多次 WriteTo。
var FrameObjPool *framePool

func init() {
	FrameObjPool = &framePool{
		pool: sync.Pool{
			New: func() any {
				return &Frame{}
			},
		},
	}
}

// Get 从池中取出一个 Frame 并绑定 messageType 与 data（不拷贝 payload）。
func (r *framePool) Get(messageType ws.OpCode, data []byte) *Frame {
	fr := r.pool.Get().(*Frame)
	fr.set(messageType, data)
	return fr
}

// Put 将 Frame 重置后归还池；勿对 nil 调用，勿重复 Put 同一指针。
func (r *framePool) Put(fr *Frame) {
	fr.reset()
	r.pool.Put(fr)
}
