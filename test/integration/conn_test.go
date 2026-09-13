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

// 本文件覆盖连接本身与各类查询命令：
// uniqId / customerId 的列表与计数、连接信息、按 customerId 查连接、在线检查、统计信息、限流。
package integration

import (
	"encoding/hex"
	"testing"

	"netsvr/test/pkg/protocol"
)

// TestConnOpenUniqId 连接打开后业务进程会推回网关分配的 uniqId
func TestConnOpenUniqId(t *testing.T) {
	c := newWsClient(t)
	// uniqId = 网关地址(6字节) + 时间戳(4字节) + 自增(4字节)，共 14 字节，即 28 个十六进制字符
	if len(c.uniqId) != 28 {
		t.Fatalf("uniqId 长度不符合预期：期望 28，实际 %d（%s）", len(c.uniqId), c.uniqId)
	}
	if _, err := hex.DecodeString(c.uniqId); err != nil {
		t.Fatalf("uniqId 不是合法的十六进制字符串：%s", c.uniqId)
	}
}

// TestUniqIdListAndCount 网关能列出全部在线连接，且数量与列表一致
func TestUniqIdListAndCount(t *testing.T) {
	a := newWsClient(t)
	b := newWsClient(t)
	want := uniqIds(a, b)

	env := a.callOK(protocol.RouterUniqIdList, nil)
	var list uniqIdListPayload
	env.into(t, &list)
	assertContains(t, "uniqIdList", want, list.uniqIds())

	env = a.callOK(protocol.RouterUniqIdCount, nil)
	var count countPayload
	env.into(t, &count)
	if int(count.Count) != len(list.uniqIds()) {
		t.Fatalf("uniqIdCount 与 uniqIdList 不一致：count=%d，list=%d", count.Count, len(list.uniqIds()))
	}
}

// TestCustomerIdListAndCount 登录后的 customerId 能被列出，且数量与列表一致
func TestCustomerIdListAndCount(t *testing.T) {
	a := newWsClient(t)
	b := newWsClient(t)
	signInAs(t, a, userXuanDe)
	signInAs(t, b, userYunChang)
	want := []string{userXuanDeId, userYunChangId}

	env := a.callOK(protocol.RouterCustomerIdList, nil)
	var list customerIdListPayload
	env.into(t, &list)
	assertContains(t, "customerIdList", want, list.all())

	env = a.callOK(protocol.RouterCustomerIdCount, nil)
	var count customerIdCountPayload
	env.into(t, &count)
	if int(count.total()) != len(list.all()) {
		t.Fatalf("customerIdCount 与 customerIdList 不一致：count=%d，list=%d", count.total(), len(list.all()))
	}
}

// TestCheckOnline 在线检查只返回真实在线的 uniqId
func TestCheckOnline(t *testing.T) {
	a := newWsClient(t)
	b := newWsClient(t)

	env := a.callOK(protocol.RouterCheckOnline, map[string]any{
		"uniqIds": []string{a.uniqId, b.uniqId, "0000000000000000000000000000"},
	})
	var payload uniqIdListPayload
	env.into(t, &payload)
	assertSameSet(t, "checkOnline", uniqIds(a, b), payload.uniqIds())
}

// TestConnInfoByCustomerId 能按 customerId 取到其名下所有连接的信息
func TestConnInfoByCustomerId(t *testing.T) {
	a := newWsClient(t)
	b := newWsClient(t)
	signInAs(t, a, userXuanDe)
	signInAs(t, b, userYunChang)

	env := a.callOK(protocol.RouterConnInfoByCustomerId, map[string]any{
		"customerIds": []string{userXuanDeId, userYunChangId},
	})
	var payload connInfoByCustomerIdPayload
	env.into(t, &payload)

	assertSameSet(t, "玄德的连接", []string{a.uniqId}, payload.customerUniqIds(userXuanDeId))
	assertSameSet(t, "云长的连接", []string{b.uniqId}, payload.customerUniqIds(userYunChangId))

	// 不存在的 customerId 不应有任何连接
	if got := payload.customerUniqIds("不存在的customerId"); len(got) != 0 {
		t.Fatalf("不存在的 customerId 不应有连接，实际 %v", got)
	}
}

// TestMetrics 网关的统计信息能被取到，且已计入本次产生的连接打开次数
func TestMetrics(t *testing.T) {
	a := newWsClient(t)
	env := a.callOK(protocol.RouterMetrics, nil)
	var metrics metricsPayload
	env.into(t, &metrics)
	if len(metrics) == 0 {
		t.Fatalf("统计信息不应为空")
	}
	var found bool
	for _, item := range metrics {
		if item.Description == "" {
			t.Fatalf("统计项缺少描述：%+v", item)
		}
		if item.Description == "客户连接的打开次数" && item.Count > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("应统计到客户连接的打开次数，实际：%+v", metrics)
	}
}

// TestLimit 限流配置可读可写：下发的新值会被网关采纳并回显。
// 注意协议里 onOpen/onMessage 只有大于 0 才会生效（0 不会关闭限流器），
// 因此这里读出基线后再改成新值，最后恢复成基线，而不是设成 0。
func TestLimit(t *testing.T) {
	c := newWsClient(t)

	env := c.callOK(protocol.RouterLimit, map[string]any{"onOpen": 0, "onMessage": 0})
	var baseline limitPayload
	env.into(t, &baseline)
	base, ok := baseline[taskAddr]
	if !ok {
		t.Fatalf("限流回包里没有网关 %s 的配置：%s", taskAddr, env.Data)
	}
	// 恢复成基线，避免影响其它用例
	defer func() {
		c.callOK(protocol.RouterLimit, map[string]any{"onOpen": base.OnOpen, "onMessage": base.OnMessage})
	}()

	const onOpen, onMessage = 123, 456
	if base.OnOpen == onOpen || base.OnMessage == onMessage {
		t.Fatalf("集成测试环境的限流基线不应与待写入的值相同：%+v", base)
	}
	env = c.callOK(protocol.RouterLimit, map[string]any{"onOpen": onOpen, "onMessage": onMessage})
	var payload limitPayload
	env.into(t, &payload)
	got, ok := payload[taskAddr]
	if !ok {
		t.Fatalf("限流回包里没有网关 %s 的配置：%s", taskAddr, env.Data)
	}
	if got.OnOpen != onOpen || got.OnMessage != onMessage {
		t.Fatalf("限流配置回显不符合预期：期望 onOpen=%d onMessage=%d，实际 onOpen=%d onMessage=%d",
			onOpen, onMessage, got.OnOpen, got.OnMessage)
	}
}
