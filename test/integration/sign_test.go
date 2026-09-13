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

// 本文件覆盖登录、退出登录、伪造登录与伪造退出登录。
package integration

import (
	"testing"

	"netsvr/test/pkg/protocol"
)

// TestSignInAndSignOut 登录会把 customerId/session/topics 写入网关，退出登录会把它们清空
func TestSignInAndSignOut(t *testing.T) {
	c := newWsClient(t)

	info := signInAs(t, c, userXuanDe)
	if info.Name != userXuanDe {
		t.Fatalf("登录回包的用户名不符合预期：期望 %s，实际 %s", userXuanDe, info.Name)
	}
	if info.UniqId != c.uniqId {
		t.Fatalf("登录回包的 uniqId 不符合预期：期望 %s，实际 %s", c.uniqId, info.UniqId)
	}

	item, ok := connInfoOf(t, c, c.uniqId)
	if !ok {
		t.Fatalf("登录后应能查到自己的连接信息")
	}
	if item.CustomerId != userXuanDeId {
		t.Fatalf("登录后的 customerId 不符合预期：期望 %s，实际 %s", userXuanDeId, item.CustomerId)
	}
	if item.Session == "" {
		t.Fatalf("登录后的 session 不应为空")
	}
	assertSameSet(t, "登录后的主题", []string{"桃园结义", "小品频道"}, item.Topics)

	env := c.callOK(protocol.RouterSignOut, nil)
	if env.Message != "退出登录成功" {
		t.Fatalf("退出登录回包不符合预期：%s", env.Message)
	}

	item, ok = connInfoOf(t, c, c.uniqId)
	if !ok {
		t.Fatalf("退出登录后应仍能查到自己的连接信息")
	}
	if item.CustomerId != "" {
		t.Fatalf("退出登录后 customerId 应为空，实际 %s", item.CustomerId)
	}
	if item.Session != "" {
		t.Fatalf("退出登录后 session 应为空，实际 %s", item.Session)
	}
	if len(item.Topics) != 0 {
		t.Fatalf("退出登录后主题应为空，实际 %v", item.Topics)
	}
}

// TestSignInFail 账号或密码错误时返回失败码，且不写入网关
func TestSignInFail(t *testing.T) {
	c := newWsClient(t)
	env := c.call(protocol.RouterSignIn, map[string]any{
		"username": userXuanDe,
		"password": "错误的密码",
	})
	if env.Code == 0 {
		t.Fatalf("密码错误时应返回失败码")
	}
	item, ok := connInfoOf(t, c, c.uniqId)
	if !ok {
		t.Fatalf("应能查到自己的连接信息")
	}
	if item.CustomerId != "" {
		t.Fatalf("登录失败不应写入 customerId，实际 %s", item.CustomerId)
	}
}

// TestSignInForForge 伪造登录会按参数写入随机的 customerId、session 与主题，伪造退出会清空它们
func TestSignInForForge(t *testing.T) {
	c := newWsClient(t)
	const topicNum, sessionLen = 3, 16

	env := c.callOK(protocol.RouterSignInForForge, map[string]any{
		"topicNum":   topicNum,
		"sessionLen": sessionLen,
	})
	var info clientInfoPayload
	env.into(t, &info)
	if info.Id == "" {
		t.Fatalf("伪造登录回包缺少客户 id：%s", env.Data)
	}

	item, ok := connInfoOf(t, c, c.uniqId)
	if !ok {
		t.Fatalf("伪造登录后应能查到自己的连接信息")
	}
	if item.CustomerId != info.Id {
		t.Fatalf("伪造登录的 customerId 不符合预期：期望 %s，实际 %s", info.Id, item.CustomerId)
	}
	if len(item.Session) != sessionLen {
		t.Fatalf("伪造登录的 session 长度不符合预期：期望 %d，实际 %d", sessionLen, len(item.Session))
	}
	if len(item.Topics) != topicNum {
		t.Fatalf("伪造登录的主题数量不符合预期：期望 %d，实际 %d", topicNum, len(item.Topics))
	}

	env = c.callOK(protocol.RouterSignOutForForge, nil)
	if env.Message != "退出登录成功" {
		t.Fatalf("伪造退出登录回包不符合预期：%s", env.Message)
	}
	item, ok = connInfoOf(t, c, c.uniqId)
	if !ok {
		t.Fatalf("伪造退出后应仍能查到自己的连接信息")
	}
	if item.CustomerId != "" || item.Session != "" || len(item.Topics) != 0 {
		t.Fatalf("伪造退出后连接信息应被清空，实际 customerId=%s session长度=%d topics=%v",
			item.CustomerId, len(item.Session), item.Topics)
	}
}
