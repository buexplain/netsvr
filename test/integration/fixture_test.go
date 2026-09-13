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

// 本文件是用例共用的测试数据与准备动作（登录、查连接信息、等下线、唯一命名）。
package integration

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"netsvr/test/pkg/protocol"
)

// ============================== 测试数据 ==============================

// 业务进程内置的模拟用户，见 test/business/internal/userDb
const (
	userPassword   = "123456"
	userXuanDe     = "玄德"
	userXuanDeId   = "玄德id-1"
	userYunChang   = "云长"
	userYunChangId = "云长id-2"
)

// noMessageTimeout 断言「收不到数据」时的等待时长
const noMessageTimeout = 300 * time.Millisecond

// notExistUniqId 一个不存在的 uniqId，用于验证「目标不存在时跳过」的语义
const notExistUniqId = "0000000000000000000000000000"

// countNamespace 保证不同用例用到的 customerId/topic 不互相干扰
var countNamespace atomic.Uint64

// uniqueName 生成一个带用例前缀的唯一名字
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, countNamespace.Add(1))
}

// uniqIds 便捷返回多个客户端的 uniqId
func uniqIds(clients ...*wsClient) []string {
	ret := make([]string, 0, len(clients))
	for _, c := range clients {
		ret = append(ret, c.uniqId)
	}
	return ret
}

// ============================== 准备动作 ==============================

// signInAs 用模拟用户登录，返回业务进程回包里的客户信息
func signInAs(t *testing.T, c *wsClient, username string) clientInfoPayload {
	t.Helper()
	env := c.callOK(protocol.RouterSignIn, map[string]any{
		"username": username,
		"password": userPassword,
	})
	var info clientInfoPayload
	env.into(t, &info)
	if info.Id == "" || info.UniqId == "" {
		t.Fatalf("登录回包缺少客户信息：%s", env.Data)
	}
	return info
}

// forgeSignIn 用伪造登录拿到一个全局唯一的 customerId
func forgeSignIn(t *testing.T, c *wsClient) string {
	t.Helper()
	env := c.callOK(protocol.RouterSignInForForge, map[string]any{"topicNum": 0, "sessionLen": 0})
	var info clientInfoPayload
	env.into(t, &info)
	if info.Id == "" {
		t.Fatalf("伪造登录未返回 customerId：%s", env.Data)
	}
	return info.Id
}

// connInfoOf 查询指定 uniqId 的连接信息
func connInfoOf(t *testing.T, observer *wsClient, uniqId string) (connInfoItem, bool) {
	t.Helper()
	env := observer.callOK(protocol.RouterConnInfo, nil)
	var payload connInfoPayload
	env.into(t, &payload)
	item, ok := payload.items()[uniqId]
	return item, ok
}

// waitOffline 轮询等待这些连接从网关下线。强制关闭是异步的，必须轮询而不能固定等待
func waitOffline(t *testing.T, observer *wsClient, targets []string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		env := observer.callOK(protocol.RouterCheckOnline, map[string]any{"uniqIds": targets})
		var payload uniqIdListPayload
		env.into(t, &payload)
		if len(payload.uniqIds()) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待连接下线超时，仍在线：%v", payload.uniqIds())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// assertOnline 断言这些连接当前都在线
func assertOnline(t *testing.T, observer *wsClient, targets []string) {
	t.Helper()
	env := observer.callOK(protocol.RouterCheckOnline, map[string]any{"uniqIds": targets})
	var payload uniqIdListPayload
	env.into(t, &payload)
	assertSameSet(t, "在线连接", targets, payload.uniqIds())
}
