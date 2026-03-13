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

package contract

import (
	"encoding/binary"
	"encoding/hex"
	"net"
	"strconv"
)

// AddrConvertToHex 将网关的task服务器监听的ip地址转为16进制字符串
func AddrConvertToHex(addr string) string {
	template := make([]byte, 6)
	//ip、port预先解析成模板
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)
	ip := net.ParseIP(host)
	//网关进程的task服务监听的ip地址
	binary.BigEndian.PutUint32(template[0:4], binary.BigEndian.Uint32(ip[12:16]))
	//网关进程的task服务监听的port
	binary.BigEndian.PutUint16(template[4:6], uint16(port))
	return hex.EncodeToString(template)
}

// UniqIdConvertToAddrAsHex 将uniqId转为网关的task服务器监听的ip地址的16进制字符串
func UniqIdConvertToAddrAsHex(uniqId string) string {
	if len(uniqId) != 28 {
		//网关分配给连接的唯一id，格式是：网关进程的task服务监听的ip地址(4字节)+网关进程的task服务监听的port(2字节)+时间戳(4字节)+自增id(4字节)，共14字节，28个16进制的字符
		return ""
	}
	return uniqId[0:12]
}
