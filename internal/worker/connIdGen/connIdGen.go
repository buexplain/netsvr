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

package connIdGen

import (
	"encoding/binary"
	"encoding/hex"
	"net"
	"netsvr/configs"
	"strconv"
	"time"
)

var template [6]byte

func init() {
	host, portStr, _ := net.SplitHostPort(configs.Config.Worker.ListenAddress)
	port, _ := strconv.Atoi(portStr)
	ip := net.ParseIP(host)
	//网关进程的worker服务监听的ip地址
	binary.BigEndian.PutUint32(template[0:4], binary.BigEndian.Uint32(ip[12:16]))
	//网关进程的worker服务监听的port
	binary.BigEndian.PutUint16(template[4:6], uint16(port))
}

// New TCP四元组可以保证当前所有连接id不会重复，时间戳可以保证不再同一秒内的连接id不重复
func New(hostPort string) string {
	b := make([]byte, 16)
	copy(b[0:6], template[:])
	host, portStr, _ := net.SplitHostPort(hostPort)
	port, _ := strconv.Atoi(portStr)
	binary.BigEndian.PutUint32(b[6:10], binary.BigEndian.Uint32(net.ParseIP(host)[12:16]))
	binary.BigEndian.PutUint16(b[10:12], uint16(port))
	binary.BigEndian.PutUint32(b[12:16], uint32(time.Now().Unix()))
	return hex.EncodeToString(b)
}
