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
	"encoding/binary"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
	"google.golang.org/protobuf/proto"
	"sync"
	"unsafe"
)

type packet struct {
	header       []byte
	body         []byte
	bodyFromPool bool
}

func (pkg *packet) reset() {
	pkg.header = pkg.header[:8]
	if pkg.bodyFromPool {
		byteslice.Put(pkg.body) //回收 packet.body
		pkg.bodyFromPool = false
	}
	pkg.body = nil
}

func (pkg *packet) set(message proto.Message, cmd netsvrProtocol.Cmd) error {
	opts := proto.MarshalOptions{}
	size := opts.Size(message)
	pooled := byteslice.Get(size)
	result, err := opts.MarshalAppend(pooled[:0], message)
	if err != nil {
		byteslice.Put(pooled) //回收 packet.body
		return err
	}
	// MarshalAppend 若换底层数组（或返回 nil 而 pooled 仍来自池），需归还 pooled；result 非池块时不可 byteslice.Put(result)
	if pooled != nil && unsafe.SliceData(result) != unsafe.SliceData(pooled) {
		byteslice.Put(pooled)
		pkg.bodyFromPool = false
	} else {
		pkg.bodyFromPool = pooled != nil
	}
	pkg.body = result
	//填充 cmd 字段 (大端序)
	binary.BigEndian.PutUint32(pkg.header[4:8], uint32(cmd))
	//填充长度字段 (大端序)
	binary.BigEndian.PutUint32(pkg.header[0:4], uint32(len(pkg.body)+4))
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
				return &packet{
					header: make([]byte, 8),
				}
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
