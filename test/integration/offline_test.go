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

// 本文件覆盖三种强制下线：按 uniqId、按 customerId、以及只针对游客（无 session）的下线。
package integration

import (
	"testing"

	"netsvr/test/pkg/protocol"
)

// forceOfflineCloseCode 强制下线使用的 websocket 关闭码
const forceOfflineCloseCode = 1008

// TestForceOffline 按 uniqId 强制下线：目标收到提示后被关闭，并最终从在线列表移除
func TestForceOffline(t *testing.T) {
	observer := newWsClient(t)
	victim := newWsClient(t)

	observer.send(protocol.RouterForceOffline, map[string]any{"uniqId": victim.uniqId})
	code, messages := victim.waitClose()
	if code != forceOfflineCloseCode {
		t.Fatalf("强制下线的关闭码不符合预期：期望 %d，实际 %d", forceOfflineCloseCode, code)
	}
	assertContains(t, "强制下线的提示", []string{"您已被迫下线！"}, messages)
	waitOffline(t, observer, []string{victim.uniqId})
}

// TestForceOfflineByCustomerId 按 customerId 强制下线：该客户名下的连接被关闭
func TestForceOfflineByCustomerId(t *testing.T) {
	observer := newWsClient(t)
	victim := newWsClient(t)
	customerId := forgeSignIn(t, victim)

	observer.send(protocol.RouterForceOfflineByCustomerId, map[string]any{"customerIds": []string{customerId}})
	code, messages := victim.waitClose()
	if code != forceOfflineCloseCode {
		t.Fatalf("强制下线的关闭码不符合预期：期望 %d，实际 %d", forceOfflineCloseCode, code)
	}
	assertContains(t, "强制下线的提示", []string{"您已被迫下线！"}, messages)
	waitOffline(t, observer, []string{victim.uniqId})
}

// TestForceOfflineGuest 只强制下线没有 session 的连接，已登录的连接不受影响
func TestForceOfflineGuest(t *testing.T) {
	observer := newWsClient(t)

	guest := newWsClient(t)
	observer.send(protocol.RouterForceOfflineGuest, map[string]any{"uniqId": guest.uniqId, "delay": 0})
	code, messages := guest.waitClose()
	if code != forceOfflineCloseCode {
		t.Fatalf("游客强制下线的关闭码不符合预期：期望 %d，实际 %d", forceOfflineCloseCode, code)
	}
	assertContains(t, "游客强制下线的提示", []string{"游客您好，您已被迫下线！"}, messages)

	// 已登录（有 session）的连接不会被关闭，也收不到任何数据
	signed := newWsClient(t)
	forgeSignIn(t, signed)
	observer.send(protocol.RouterForceOfflineGuest, map[string]any{"uniqId": signed.uniqId, "delay": 0})
	signed.assertNoMessage(noMessageTimeout)
	assertOnline(t, observer, []string{signed.uniqId})
}
