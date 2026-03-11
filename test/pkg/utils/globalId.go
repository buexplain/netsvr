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

package utils

import (
	cryptoRand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"os"
	"time"
)

// GlobalId 全局唯一id
var GlobalId string

func init() {
	var buf []byte
	//核心熵源
	randomBuf := make([]byte, 22)
	if _, err := cryptoRand.Read(randomBuf); err == nil {
		buf = append(buf, randomBuf...)
	} else {
		//随机数发生器发生错误，改为普通随机
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := range randomBuf {
			randomBuf[i] = charset[seededRand.Intn(len(charset))]
		}
		buf = append(buf, randomBuf...)
	}
	//进程id
	pidBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(pidBuf, uint16(os.Getpid()))
	buf = append(buf, pidBuf...)
	//时间戳
	ts := uint64(time.Now().UnixNano())
	tsBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBuf, ts)
	buf = append(buf, tsBuf...)
	GlobalId = hex.EncodeToString(buf)
}
