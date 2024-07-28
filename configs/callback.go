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

package configs

import "strings"

// OnOpen 连接打开时的回调函数
// 如果网关程序仅作推送消息的场景，则business进程是不存在的，在这里你可以将连接打开的信息发送到你的http api程序，并根据http api程序返回的结果决定是否拒绝客户端的连接
func OnOpen(uniqId string, messageType int8, rawQuery string, subProtocols []string, XForwardedFor string, xRealIp string, remoteAddr string) (sendToCustomerData []byte, closeConnection bool) {
	defer func() {
		// 防止panic
		_ = recover()
	}()
	if strings.Contains(rawQuery, "simplex=yes") {
		return []byte(uniqId), false
	}
	return nil, false
}

// OnClose 连接关闭时的回调函数
// 如果网关程序仅作推送消息的场景，则business进程是不存在的，该回调函数是为了方便此场景下，业务侧处理连接关闭的事件
func OnClose(uniqId string, customerId string, session string, topics []string) {
	defer func() {
		// 防止panic
		_ = recover()
	}()
}
