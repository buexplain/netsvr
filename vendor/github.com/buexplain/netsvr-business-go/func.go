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

package netsvrBusiness

import (
	"encoding/binary"
	"encoding/hex"
	"net"
	"strconv"
)

// WorkerAddrConvertToHex 将网关的worker服务器监听的地址转为16进制字符串
func WorkerAddrConvertToHex(workerAddr string) string {
	template := make([]byte, 6)
	//ip、port预先解析成模板
	host, portStr, _ := net.SplitHostPort(workerAddr)
	port, _ := strconv.Atoi(portStr)
	ip := net.ParseIP(host)
	//网关进程的worker服务监听的ip地址
	binary.BigEndian.PutUint32(template[0:4], binary.BigEndian.Uint32(ip[12:16]))
	//网关进程的worker服务监听的port
	binary.BigEndian.PutUint16(template[4:6], uint16(port))
	return hex.EncodeToString(template)
}

// UniqIdConvertToWorkerAddrAsHex 将uniqId转为网关的worker服务器监听的地址的16进制字符串
func UniqIdConvertToWorkerAddrAsHex(uniqId string) string {
	return uniqId[0:12]
}
