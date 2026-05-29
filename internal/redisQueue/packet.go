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

// Package redisQueue Redis队列
package redisQueue

import (
	"encoding/binary"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
	"google.golang.org/protobuf/proto"
	"sync"
	"unsafe"
)

type packet struct {
	message      []byte
	bodyFromPool bool
}

func (pkg *packet) reset() {
	if pkg.bodyFromPool {
		byteslice.Put(pkg.message) //回收 packet.message
		pkg.bodyFromPool = false
	}
	pkg.message = nil
}

func (pkg *packet) set(message proto.Message, cmd netsvrProtocol.Cmd) error {
	opts := proto.MarshalOptions{}
	size := opts.Size(message)
	var result []byte
	var err error
	if size > 0 {
		pooled := byteslice.Get(size + 4)
		// 将 proto 数据追加到前 4 字节之后，使 result 从偏移 0 开始包含 cmd + body
		// Redis 队列无需长度字段（list/stream 本身已有边界），仅保留 4 字节 cmd 用于消费者分派
		result, err = opts.MarshalAppend(pooled[:4], message)
		if err != nil {
			byteslice.Put(pooled) //回收 pooled 缓冲区
			return err
		}
		// MarshalAppend 若换底层数组，需归还 pooled；未换时 result 与 pooled 共享底层数组，由 reset 统一归还
		if unsafe.SliceData(result) != unsafe.SliceData(pooled) {
			byteslice.Put(pooled)
			pkg.bodyFromPool = false
		} else {
			pkg.bodyFromPool = true
		}
	} else {
		// size <= 0 时，直接序列化（byteslice.Get(0) 返回 nil，无法用于 MarshalAppend）
		// make([]byte, 4) 作为前 4 字节 cmd 占位，MarshalAppend 追加空 body
		result = make([]byte, 4)
		result, err = opts.MarshalAppend(result, message)
		if err != nil {
			return err
		}
		pkg.bodyFromPool = false
	}
	pkg.message = result
	//填充 cmd 字段 (大端序)
	binary.BigEndian.PutUint32(pkg.message[0:4], uint32(cmd))
	return nil
}

type packetPool struct {
	pool sync.Pool
}

var packetObjPool *packetPool

func init() {
	packetObjPool = &packetPool{
		pool: sync.Pool{
			New: func() any {
				return &packet{}
			},
		},
	}
}

func (r *packetPool) Get() *packet {
	return r.pool.Get().(*packet)
}

func (r *packetPool) Put(pkg *packet) {
	pkg.reset()
	r.pool.Put(pkg)
}
