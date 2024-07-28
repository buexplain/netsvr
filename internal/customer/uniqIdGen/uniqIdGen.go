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

// Package uniqIdGen 网关唯一id的生成器
package uniqIdGen

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
)

// 自增id
var incrId = readRandomUint32()

// 预先制作的ip、port模板
var template [6]byte

func init() {
	//ip、port预先解析成模板
	host, portStr, _ := net.SplitHostPort(configs.Config.Worker.ListenAddress)
	port, _ := strconv.Atoi(portStr)
	ip := net.ParseIP(host)
	//网关进程的worker服务监听的ip地址
	binary.BigEndian.PutUint32(template[0:4], binary.BigEndian.Uint32(ip[12:16]))
	//网关进程的worker服务监听的port
	binary.BigEndian.PutUint16(template[4:6], uint16(port))
}

// New
// php解码uniqId示例：
// $ret =  unpack('Nip/nport/Ntimestamp/NincrId', pack('H*', '7f00000117ad6621e43b8baa1b9a'));
// $ret['ip'] = long2ip($ret['ip']);
// var_dump($ret);
func New() string {
	b := make([]byte, 14)
	//先写入ip、port
	copy(b[0:6], template[:])
	//再写入时间戳
	binary.BigEndian.PutUint32(b[6:10], uint32(time.Now().Unix()))
	//最后写入自增id
	binary.BigEndian.PutUint32(b[10:14], atomic.AddUint32(incrId, 1))
	//转字16进制符串返回
	return hex.EncodeToString(b)
}

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
