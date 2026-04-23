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

package wsServer

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/panjf2000/gnet/v2"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"unicode/utf8"
)

type codec struct {
	preMessagePayload  *messagePayload // 上一个帧的数据
	currMessagePayload *messagePayload // 当前帧的数据
	closed             uint32          // 是否服务端主动关闭
	upgraded           bool
	compression        bool
	preMessageOpCode   ws.OpCode
	currMessageOpCode  ws.OpCode
	messageRsv         byte // 0~7 的整数，由于控制帧可以穿插在消息帧的分片中，但是控制帧又没有 rsv，所有只需一个字段即可，当控制帧穿插进来的时候，无需转移保存
}

func newCodec() codec {
	c := codec{}
	c.currMessagePayload = newMessagePayload(4)
	c.preMessageOpCode = 255
	c.currMessageOpCode = 255
	return c
}

func (r *codec) SetClosedFlagOnSafe() bool {
	return atomic.CompareAndSwapUint32(&r.closed, 0, 1)
}

func (r *codec) IsClosedOnSafe() bool {
	return atomic.LoadUint32(&r.closed) == 1
}

func (r *codec) IsCompressOnSafe() bool {
	return r.compression
}

func (r *codec) resetCurrMessage() {
	r.currMessagePayload.reset()
	if r.preMessageOpCode == 255 {
		r.currMessageOpCode = 255
		r.messageRsv = 0
	} else {
		r.currMessageOpCode = r.preMessageOpCode
		r.currMessagePayload = r.preMessagePayload
		r.preMessageOpCode = 255
		r.preMessagePayload = nil
	}
}

func (r *codec) upgrade(onUpgradeCheck func(req *http.Request) *http.Response, c gnet.Conn) (*http.Request, gnet.Action) {
	peek, err := c.Peek(-1)
	if err != nil {
		return nil, gnet.Close
	}
	//判断 http head 是否完整
	if l := len(peek); l < 4 || bytes.Equal(peek[l-4:], []byte("\r\n\r\n")) == false {
		return nil, gnet.None
	}

	//读取 http head
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(peek)))
	if err != nil {
		//返回 HTTP 400
		resp := http.Response{
			StatusCode: http.StatusBadRequest,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader(http.StatusText(http.StatusBadRequest))),
		}
		_ = resp.Write(c)
		return nil, gnet.Close
	}

	//判断请求方法
	if req.Method != "GET" {
		//响应 HTTP 405
		resp := http.Response{
			StatusCode: http.StatusMethodNotAllowed,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader(http.StatusText(http.StatusMethodNotAllowed))),
		}
		_ = resp.Write(c)
		return nil, gnet.Close
	}

	//没有Upgrade头，说明无须升级协议，响应一个正常的http ok
	if _, ok := req.Header["Upgrade"]; ok == false {
		resp := http.Response{
			StatusCode: http.StatusOK,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader("Hello netsvr")),
		}
		_ = resp.Write(c)
		return nil, gnet.Close
	}

	//握手之前验证
	if resp := onUpgradeCheck(req); resp != nil {
		_ = resp.Write(c)
		return nil, gnet.Close
	}

	//验证 Upgrade 头
	if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") == false {
		resp := http.Response{
			StatusCode: http.StatusBadRequest,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader("invalid Upgrade header")),
		}
		_ = resp.Write(c)
		return nil, gnet.Close
	}

	//验证 Connection 头包含 "Upgrade"
	if connection := req.Header.Get("Connection"); connection != "Upgrade" && connection != "upgrade" {
		resp := http.Response{
			StatusCode: http.StatusBadRequest,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader("invalid Connection header")),
		}
		_ = resp.Write(c)
		return nil, gnet.Close
	}

	//验证 Sec-WebSocket-Version == 13
	if req.Header.Get("Sec-WebSocket-Version") != "13" {
		// 返回 426 Upgrade Required + 正确版本
		resp := http.Response{
			StatusCode: http.StatusUpgradeRequired,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header: http.Header{
				"Sec-WebSocket-Version": []string{"13"},
			},
			Body: io.NopCloser(strings.NewReader(http.StatusText(http.StatusUpgradeRequired))),
		}
		_ = resp.Write(c)
		return nil, gnet.Close
	}

	//协商压缩扩展
	if ext := req.Header.Get("Sec-WebSocket-Extensions"); ext != "" {
		options, ok := httphead.ParseOptions([]byte(ext), nil)
		if ok == false {
			resp := http.Response{
				StatusCode: http.StatusBadRequest,
				ProtoMajor: 1,
				ProtoMinor: 1,
				Body:       io.NopCloser(strings.NewReader("invalid Sec-WebSocket-Extensions header")),
			}
			_ = resp.Write(c)
			return nil, gnet.Close
		}
		for _, option := range options {
			if bytes.Equal(option.Name, wsflate.ExtensionNameBytes) {
				r.compression = true
				break
			}
		}
	}

	//获取 Sec-WebSocket-Key
	key := req.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		resp := http.Response{
			StatusCode: http.StatusBadRequest,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader("missing Sec-WebSocket-Key")),
		}
		_ = resp.Write(c)
		return nil, gnet.Close
	}

	//计算 Sec-WebSocket-Accept
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

	//构造 101 Switching Protocols 响应
	resp := http.Response{
		StatusCode: http.StatusSwitchingProtocols,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Upgrade":              []string{"websocket"},
			"Connection":           []string{"Upgrade"},
			"Sec-WebSocket-Accept": []string{acceptKey},
		},
	}

	//添加压缩扩展
	if r.compression {
		//两个参数直接决定了系统的内存开销和压缩率，除非你的并发连接数非常少，且对压缩率有极致要求，否则永远选择 no_context_takeover。
		//这是一个典型的用少量性能损失换取巨大可伸缩性和稳定性的架构决策。
		// server_no_context_takeover //告诉客户端，服务器不会为客户端的不同消息复用同一个 LZ77 滑动窗口（即压缩上下文），每次压缩都是独立的。
		// client_no_context_takeover 告诉客户端，它在解压缩来自服务器的消息时，也不应该复用解压上下文。
		resp.Header["sec-websocket-extensions"] = []string{"permessage-deflate; server_no_context_takeover; client_no_context_takeover"}
	}

	//清除所有请求数据，有些请求会在get中添加数据，所以需要清空
	_, err = c.Discard(c.InboundBuffered())
	if err != nil {
		return nil, gnet.Close
	}

	//发送响应
	err = resp.Write(c)
	if err != nil {
		return nil, gnet.Close
	}

	//升级成功
	r.upgraded = true

	return req, gnet.None
}

func (r *codec) decode(c gnet.Conn) ([]byte, ws.StatusCode, error) {
	for {
		peek, err := c.Peek(-1)
		if err != nil {
			if errors.Is(err, io.ErrShortBuffer) {
				//数据不完整
				return nil, ws.StatusCode(0), io.ErrShortBuffer
			}
			return nil, ws.StatusInternalServerError, err
		}
		if len(peek) < 2 {
			return nil, ws.StatusCode(0), io.ErrShortBuffer
		}
		//ws.ReadHeader()
		var head ws.Header
		head.Fin = peek[0]&0x80 != 0
		head.Rsv = (peek[0] & 0x70) >> 4
		head.OpCode = ws.OpCode(peek[0] & 0x0f)
		validOp := head.OpCode == ws.OpContinuation ||
			head.OpCode == ws.OpText ||
			head.OpCode == ws.OpBinary ||
			head.OpCode == ws.OpClose ||
			head.OpCode == ws.OpPing ||
			head.OpCode == ws.OpPong
		if !validOp {
			return nil, ws.StatusProtocolError, errors.New("unsupported opcode")
		}
		if head.Rsv != 0 {
			//未协商开启压缩扩展，RSV1 bits MUST be 0
			if r.compression == false {
				return nil, ws.StatusProtocolError, errors.New("RSV1 bits must be 0 without extensions")
			} else if head.OpCode.IsControl() {
				//控制帧的 RSV1 不对
				return nil, ws.StatusProtocolError, errors.New("RSV1 must be 0")
			}
		}
		//已经得到了首帧，此刻的是后续帧
		if r.currMessageOpCode != 255 {
			//控制帧不允许分片
			if r.currMessageOpCode.IsControl() {
				return nil, ws.StatusProtocolError, errors.New("control message MUST NOT be fragmented")
			}
			//后续帧
			if head.OpCode.IsControl() {
				//控制帧穿插在数据帧分片之间，保存之前的数据帧
				r.preMessageOpCode = r.currMessageOpCode
				r.preMessagePayload = r.currMessagePayload
				r.currMessageOpCode = 255
				r.currMessagePayload = newMessagePayload(1)
			} else {
				//数据帧的连续帧的 opcode 不对
				if head.OpCode != ws.OpContinuation {
					return nil, ws.StatusProtocolError, errors.New("non-first fragment must be continuation")
				}
				//协商已开启压缩扩展，连续帧的 RSV1 不对
				if r.compression && head.Rsv != 0 {
					return nil, ws.StatusProtocolError, errors.New("RSV1 must be 0")
				}
			}
		}
		if peek[1]&0x80 == 0 {
			return nil, ws.StatusProtocolError, errors.New("client must mask data")
		}
		var extra = 0
		length := peek[1] & 0x7f
		switch {
		case length < 126:
			head.Length = int64(length)
			extra = 4 // 2 bytes header + 4 bytes mask
		case length == 126:
			extra = 6 // 2 bytes header + 2 bytes length + 4 bytes mask
		case length == 127:
			extra = 12 // 2 bytes header + 8 bytes length + 4 bytes mask
		default:
			return nil, ws.StatusProtocolError, errors.New("unexpected payload length bits")
		}
		if len(peek) < 2+extra {
			//数据不完整
			return nil, ws.StatusCode(0), io.ErrShortBuffer
		}
		peek = peek[2:]
		switch {
		case length == 126:
			head.Length = int64(binary.BigEndian.Uint16(peek[:2]))
			peek = peek[2:]
		case length == 127:
			if peek[0]&0x80 != 0 {
				return nil, ws.StatusProtocolError, errors.New("the most significant bit must be 0")
			}
			head.Length = int64(binary.BigEndian.Uint64(peek[:8]))
			peek = peek[8:]
		}
		//校验 Ping/Pong/CloseOnSafe 的 payload 长度
		if head.OpCode.IsControl() && head.Length > 125 {
			return nil, ws.StatusProtocolError, errors.New("control frame too long")
		}
		if len(peek) < (int)(head.Length)+4 {
			//数据不完整
			return nil, ws.StatusCode(0), io.ErrShortBuffer
		}
		copy(head.Mask[:], peek[:4])
		//即将得到一个完整的消息帧，此刻可以记住首帧的opcode、RSV
		if r.currMessageOpCode == 255 {
			if r.compression && head.Rsv != 4 && head.Rsv != 0 {
				//协商启用了压缩扩展，首帧的 RSV1 不对
				return nil, ws.StatusProtocolError, errors.New("RSV1 must be 0 or 4")
			}
			r.messageRsv = head.Rsv
			//在没有待继续消息时，禁止接收 Continuation 帧
			if head.OpCode == ws.OpContinuation {
				return nil, ws.StatusProtocolError, errors.New("unexpected continuation frame: no message to continue")
			}
			r.currMessageOpCode = head.OpCode
		}
		r.currMessagePayload.append(peek[4:4+(int)(head.Length)], head.Mask)
		_, err = c.Discard(2 + extra + (int)(head.Length))
		if err != nil {
			return nil, ws.StatusInternalServerError, err
		}
		if head.Fin {
			completeMessagePayload := r.currMessagePayload.merge()
			//当前 header 已经是一个完整消息
			if r.currMessageOpCode == ws.OpText {
				//解压缩数据
				if r.compression && r.messageRsv == 4 {
					completeMessagePayload, err = wsflate.DefaultHelper.Decompress(completeMessagePayload)
					if err != nil {
						return nil, ws.StatusInvalidFramePayloadData, errors.New("invalid deflate stream")
					}
				}
				//文本消息，校验utf8
				if utf8.Valid(completeMessagePayload) == false {
					return nil, ws.StatusInvalidFramePayloadData, errors.New("invalid utf8 in text message")
				}
			} else if r.currMessageOpCode == ws.OpBinary {
				//解压缩数据
				if r.compression && r.messageRsv == 4 {
					completeMessagePayload, err = wsflate.DefaultHelper.Decompress(completeMessagePayload)
					if err != nil {
						return nil, ws.StatusInvalidFramePayloadData, errors.New("invalid deflate stream")
					}
				}
			} else if r.currMessageOpCode == ws.OpClose && len(completeMessagePayload) > 0 {
				//关闭消息，校验关闭码
				pl := len(completeMessagePayload)
				if pl == 1 || pl > 125 {
					return nil, ws.StatusProtocolError, errors.New("close frame payload length invalid")
				}
				if pl >= 2 {
					code := ws.StatusCode(binary.BigEndian.Uint16(completeMessagePayload[:2]))
					invalidCode := code.In(ws.StatusRangeNotInUse) ||
						code == ws.StatusNoMeaningYet ||
						code == ws.StatusNoStatusRcvd ||
						code == ws.StatusAbnormalClosure ||
						code == 1016 ||
						code == 1100 ||
						code == 2000 ||
						code == 2999
					if invalidCode {
						return nil, ws.StatusProtocolError, errors.New("invalid close status code")
					}
					if pl > 2 && utf8.Valid(completeMessagePayload[2:]) == false {
						return nil, ws.StatusInvalidFramePayloadData, errors.New("invalid utf8 in text message")
					}
				}
			}
			return completeMessagePayload, ws.StatusCode(0), nil
		}
	}
}
