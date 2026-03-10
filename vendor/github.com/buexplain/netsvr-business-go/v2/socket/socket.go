/**
* Copyright 2024 buexplain@qq.com
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

package socket

import (
	"bufio"
	"encoding/binary"
	"errors"
	"github.com/buexplain/netsvr-business-go/v2/log"
	"io"
	"net"
	"sync/atomic"
	"time"
)

type Socket struct {
	addr           string
	receiveTimeout time.Duration
	sendTimeout    time.Duration
	connectTimeout time.Duration
	socket         net.Conn
	socketBufIO    *bufio.Reader
	connected      int32
}

const socketConnectedNo = 0
const socketConnectIng = 1
const socketConnectedYes = 2

func New(addr string, receiveTimeout time.Duration, sendTimeout time.Duration, connectTimeout time.Duration) *Socket {
	return &Socket{
		addr:           addr,
		receiveTimeout: receiveTimeout,
		sendTimeout:    sendTimeout,
		connectTimeout: connectTimeout,
		connected:      0,
	}
}

func (s *Socket) GetAddr() string {
	return s.addr
}

func (s *Socket) IsConnected() bool {
	return atomic.LoadInt32(&s.connected) == socketConnectedYes
}

func (s *Socket) close() {
	if atomic.CompareAndSwapInt32(&s.connected, socketConnectedYes, socketConnectedNo) {
		_ = s.socket.Close()
	}
}

func (s *Socket) Close() {
	if atomic.CompareAndSwapInt32(&s.connected, socketConnectIng, socketConnectedNo) {
		return
	}
	s.close()
}

func (s *Socket) Connect() bool {
	if atomic.CompareAndSwapInt32(&s.connected, socketConnectedNo, socketConnectIng) {
		defer atomic.CompareAndSwapInt32(&s.connected, socketConnectIng, socketConnectedNo)
		d := net.Dialer{
			Timeout: s.connectTimeout,
		}
		conn, err := d.Dial("tcp", s.addr)
		if err != nil {
			log.Info("connect to "+s.addr+" failed", "error", err)
			return false
		}
		if atomic.CompareAndSwapInt32(&s.connected, socketConnectIng, socketConnectedYes) {
			s.socket = conn
			s.socketBufIO = bufio.NewReaderSize(conn, 65536)
			return true
		} else {
			_ = conn.Close()
		}
	}
	return false
}

func (s *Socket) Send(message []byte) bool {
	totalLen := len(message)
	data := make([]byte, totalLen+4)
	binary.BigEndian.PutUint32(data[0:4], uint32(totalLen))
	totalLen += 4
	copy(data[4:totalLen], message)
	var err error
	var writeLen int
	//写入到连接中
	for {
		//设置写超时
		var timeout time.Time
		if s.sendTimeout > 0 {
			timeout = time.Now().Add(s.sendTimeout)
		} else {
			timeout = time.Time{}
		}
		if err = s.socket.SetWriteDeadline(timeout); err != nil {
			if s.IsConnected() {
				log.Info("set write timeout failed", "error", err)
			}
			return false
		}
		//写入数据
		writeLen, err = s.socket.Write(data)
		//写入成功
		if err == nil {
			//写入成功
			if writeLen == len(data) {
				return true
			}
			//短写，继续写入
			data = data[writeLen:]
			time.Sleep(time.Millisecond * 10)
			continue
		}
		//写入错误
		//没有写入任何数据，tcp管道未被污染，丢弃本次数据，并打印日志
		var opErr *net.OpError
		if errors.As(err, &opErr) && opErr.Timeout() && totalLen == len(data[writeLen:]) {
			if s.IsConnected() {
				log.Info("send message to "+s.addr+" timeout", "error", err)
			}
			return false
		}
		//写入过部分数据，tcp管道已污染，对端已经无法拆包，必须关闭连接
		s.close()
		return false
	}
}

func (s *Socket) Receive() []byte {
	var timeout time.Time
	if s.receiveTimeout > 0 {
		timeout = time.Now().Add(s.receiveTimeout)
	} else {
		timeout = time.Time{}
	}
	if err := s.socket.SetReadDeadline(timeout); err != nil {
		if s.IsConnected() {
			s.close()
			log.Info("set read timeout failed", "error", err)
		}
		return nil
	}
	data := make([]byte, 4)
	if _, err := io.ReadFull(s.socketBufIO, data); err != nil {
		if s.IsConnected() {
			s.close()
			log.Info("read message length from "+s.addr+" failed", "error", err)
		}
		return nil
	}
	dataLen := binary.BigEndian.Uint32(data)
	data = make([]byte, dataLen)
	if _, err := io.ReadAtLeast(s.socketBufIO, data, int(dataLen)); err != nil {
		if s.IsConnected() {
			s.close()
			log.Info("read message from "+s.addr+" failed", "error", err)
		}
		return nil
	}
	return data
}
