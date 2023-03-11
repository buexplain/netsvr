/**
* Copyright 2022 buexplain@qq.com
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

package utils

import (
	"encoding/json"
	"math/rand"
	"netsvr/test/pkg/protocol"
	"unsafe"
)

// NewResponse 构造一个返回给客户端响应
func NewResponse(cmd protocol.Cmd, data interface{}) []byte {
	tmp := map[string]interface{}{"cmd": cmd, "data": data}
	ret, _ := json.Marshal(tmp)
	return ret
}

// StrToReadOnlyBytes 字符串无损转字节切片，转换后的切片，不能做修改操作，因为go的字符串是不可修改的
func StrToReadOnlyBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// BytesToReadOnlyString 字节切片无损转字符串
func BytesToReadOnlyString(bt []byte) string {
	if len(bt) == 0 {
		return ""
	}
	return unsafe.String(&bt[0], len(bt))
}

var charPool = [36]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z'}

// GetRandStr n=4有1413720种排列 n=3有42840种排列，n=2有1260种排列
func GetRandStr(n int) string {
	s := make([]byte, 0, n)
	for {
		n--
		if n < 0 {
			break
		}
		s = append(s, charPool[rand.Intn(35)])
	}
	return string(s)
}
