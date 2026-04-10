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

// Package n 网关唯一id的生成器
package n

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"netsvr/configs"
	"netsvr/internal/log"
	"os"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"
)

// 自增id
var incrId = readRandomUint32()

// 预计算的 template hex 字符串（6字节 -> 12字符）
var templateHex [12]byte

// hexDigits 用于快速十六进制转换
const hexDigits = "0123456789abcdef"

func init() {
	//ip、port预先解析成模板
	host, portStr, _ := net.SplitHostPort(configs.Config.Task.ListenAddress)
	//host, portStr, _ := net.SplitHostPort("127.0.0.1:6060")
	port, _ := strconv.Atoi(portStr)
	ip := net.ParseIP(host)
	var template [6]byte
	//网关进程的worker服务监听的ip地址
	binary.BigEndian.PutUint32(template[0:4], binary.BigEndian.Uint32(ip[12:16]))
	//网关进程的worker服务监听的port
	binary.BigEndian.PutUint16(template[4:6], uint16(port))
	//预计算 hex 形式
	hex.Encode(templateHex[:], template[:])
}

// New
// 网关分配给连接的唯一id，格式是：网关进程的task服务监听的ip地址(4字节)+网关进程的task服务监听的port(2字节)+时间戳(4字节)+自增id(4字节)，共14字节，28个16进制的字符
// php解码uniqId示例：
// $ret =  unpack('Nip/nport/Ntimestamp/NincrId', pack('H*', '7f00000117ad6621e43b8baa1b9a'));
// $ret['ip'] = long2ip($ret['ip']);
// var_dump($ret);
func New() string {
	b := make([]byte, 28) // 12(templateHex) + 16(timestamp+incrId)

	// 复制预计算的 template hex
	copy(b[0:12], templateHex[:])

	// 获取当前时间戳和自增ID
	timestamp := uint32(time.Now().Unix())
	incr := atomic.AddUint32(incrId, 1)

	// 编码 timestamp (4字节 -> 8字符)
	encodeUint32ToHex(b[12:20], timestamp)
	// 编码 incrId (4字节 -> 8字符)
	encodeUint32ToHex(b[20:28], incr)

	// 返回字符串
	return unsafe.String(unsafe.SliceData(b), 28)
}

// encodeUint32ToHex 将 uint32 编码为 8 字符的 hex 字符串（大端序）
func encodeUint32ToHex(dst []byte, val uint32) {
	// 从高位到低位逐个字节处理
	for i := 0; i < 4; i++ {
		b := byte(val >> (24 - i*8))   // 提取每个字节
		dst[i*2] = hexDigits[b>>4]     // 高4位
		dst[i*2+1] = hexDigits[b&0x0f] // 低4位
	}
}

// readRandomUint32 读取随机 uint32
func readRandomUint32() *uint32 {
	var b [4]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		log.Logger.Error().Err(err).Msg("cannot initialize uniqIdGen package with crypto.rand.Reader")
		os.Exit(1)
	}
	i := (uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24)
	return &i
}
