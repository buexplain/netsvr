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
	"github.com/panjf2000/gnet/v2"
	"net"
)

// Conn 封装gnet.Conn, 添加了Info和Codec
type Conn struct {
	codec
	info
	conn gnet.Conn
}

func (r *Conn) CloseOnSafe() {
	_ = r.conn.Close()
}

func (r *Conn) AsyncWriteOnSafe(buf []byte) error {
	return r.conn.AsyncWrite(buf, nil)
}

func (r *Conn) RemoteAddrOnSafe() net.Addr {
	return r.conn.RemoteAddr()
}

func NewConn(conn gnet.Conn) *Conn {
	return &Conn{
		codec: newCodec(),
		info:  newInfo(),
		conn:  conn,
	}
}
