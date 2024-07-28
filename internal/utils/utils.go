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
	"net"
	"net/http"
	"os"
	"strings"
	"unsafe"
)

// ParseSubProtocols 解析websocket的子协议
func ParseSubProtocols(r *http.Request) []string {
	h := strings.TrimSpace(r.Header.Get("Sec-Websocket-Protocol"))
	if h == "" {
		return nil
	}
	protocols := strings.Split(h, ",")
	for i := range protocols {
		protocols[i] = strings.TrimSpace(protocols[i])
	}
	return protocols
}

// StrToReadOnlyBytes 字符串无损转字节切片，转换后的切片，不能做修改操作，因为go的字符串是不可修改的
func StrToReadOnlyBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// GetHostByName 返回主机名 name 对应的 IPv4 互联网地址。成功时返回 IPv4 地址，失败时原封不动返回 name 字符串。
func GetHostByName(name string) string {
	addrList, err := net.LookupHost(name)
	if err != nil {
		return name
	}
	global := name
	loopBack := name
	unspecified := name
	for _, addr := range addrList {
		ip := net.ParseIP(addr)
		if ip.IsGlobalUnicast() {
			global = ip.To4().String()
		}
		if ip.IsLoopback() {
			loopBack = ip.To4().String()
		}
		if ip.IsUnspecified() {
			unspecified = "127.0.0.1"
		}
	}
	if global != name {
		return global
	}
	if loopBack != name {
		return loopBack
	}
	return unspecified
}

// IsValidIPv4 判断是否为有效的 IPv4 地址
func IsValidIPv4(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		return parsedIP.To4() != nil
	}
	return false
}

// GetLocalIPAddress 获取第一个非回环的IPv4地址
func GetLocalIPAddress() string {
	addrList, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrList {
		ip, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() {
			continue
		}
		ipV4 := ip.IP.To4()
		if ipV4 != nil {
			return ipV4.String()
		}
	}
	return ""
}

// IsFile 判断给定的文件名是否对应于一个存在的文件。
// 如果文件存在且不是目录，则返回 true。
// 如果文件不存在，则返回 false。
// 如果出现其他错误，则返回错误信息。
func IsFile(filename string) (bool, error) {
	fi, err := os.Stat(filename)
	if err == nil {
		if fi.IsDir() {
			return false, nil
		}
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
