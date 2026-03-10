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
	"github.com/gobwas/ws"
	"github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
)

func NewMessagePayload(len int) *MessagePayload {
	return &MessagePayload{
		fragments: make([][]byte, 0, len),
	}
}

type MessagePayload struct {
	fragments [][]byte
}

func (r *MessagePayload) reset() {
	// 解除分片引用，避免内存滞留
	for i, frag := range r.fragments {
		if i > 0 {
			byteslice.Put(frag)
		}
		r.fragments[i] = nil
	}
	if len(r.fragments) > 16 {
		r.fragments = make([][]byte, 0, 4)
	} else {
		r.fragments = r.fragments[:0] // 保留容量复用
	}
}

func (r *MessagePayload) append(fragment []byte, mask [4]byte) {
	var tmp []byte
	if len(r.fragments) == 0 {
		tmp = make([]byte, len(fragment))
	} else {
		tmp = byteslice.Get(len(fragment))
	}
	copy(tmp, fragment)
	ws.Cipher(tmp, mask, 0)
	r.fragments = append(r.fragments, tmp)
}

func (r *MessagePayload) merge() []byte {
	n := len(r.fragments)
	if n == 0 {
		return nil
	}

	if n == 1 {
		return r.fragments[0] // 安全：append 时已复制
	}

	// 计算总长度
	totalLen := 0
	for _, frag := range r.fragments {
		totalLen += len(frag)
	}
	// 一次性分配 + 合并
	ret := make([]byte, totalLen)
	offset := 0
	for _, frag := range r.fragments {
		copy(ret[offset:], frag)
		offset += len(frag)
	}
	return ret
}
