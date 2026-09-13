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

// 本文件覆盖单播、按 customerId 单播、批量单播、组播与广播这些「投递给客户」的命令。
package integration

import (
	"testing"

	"netsvr/test/pkg/protocol"
)

// TestSingleCast 按 uniqId 单播：只有目标连接收到
func TestSingleCast(t *testing.T) {
	sender := newWsClient(t)
	target := newWsClient(t)
	other := newWsClient(t)

	message := uniqueName("singleCast")
	sender.send(protocol.RouterSingleCast, map[string]any{"message": message, "uniqId": target.uniqId})

	target.expectCast(protocol.RouterSingleCast, fromUserMessage{
		Message:  message,
		FromUser: "uniqId(" + sender.uniqId + ")",
	})
	other.assertNoMessage(noMessageTimeout)
}

// TestSingleCastByCustomerId 按 customerId 单播：该客户名下的连接收到
func TestSingleCastByCustomerId(t *testing.T) {
	sender := newWsClient(t)
	target := newWsClient(t)
	customerId := forgeSignIn(t, target)
	other := newWsClient(t)

	message := uniqueName("singleCastByCustomerId")
	sender.send(protocol.RouterSingleCastByCustomerId, map[string]any{"message": message, "customerId": customerId})

	target.expectCast(protocol.RouterSingleCastByCustomerId, fromUserMessage{
		Message:  message,
		FromUser: "uniqId(" + sender.uniqId + ")",
	})
	other.assertNoMessage(noMessageTimeout)
}

// TestSingleCastBulk 按 uniqId 批量单播：多条消息按顺序一一对应多个目标
func TestSingleCastBulk(t *testing.T) {
	sender := newWsClient(t)
	first := newWsClient(t)
	second := newWsClient(t)
	other := newWsClient(t)

	messageFirst := uniqueName("bulkFirst")
	messageSecond := uniqueName("bulkSecond")
	sender.send(protocol.RouterSingleCastBulk, map[string]any{
		"message": []string{messageFirst, messageSecond},
		"uniqIds": []string{first.uniqId, second.uniqId},
	})

	wantFrom := "uniqId(" + sender.uniqId + ")"
	first.expectCast(protocol.RouterSingleCastBulk, fromUserMessage{Message: messageFirst, FromUser: wantFrom})
	second.expectCast(protocol.RouterSingleCastBulk, fromUserMessage{Message: messageSecond, FromUser: wantFrom})
	other.assertNoMessage(noMessageTimeout)
}

// TestSingleCastBulkSkipInvalidTarget 网关会跳过「目标不存在」与「数据为空」两类数据项，且不影响同批里的正常投递
func TestSingleCastBulkSkipInvalidTarget(t *testing.T) {
	sender := newWsClient(t)
	target := newWsClient(t)
	other := newWsClient(t)

	validData := uniqueName("bulkValid")
	// 三个数据项依次对应：正常投递、目标不存在、数据为空
	sender.send(protocol.RouterSingleCastBulk, map[string]any{
		"message": []string{validData, uniqueName("bulkNotExistTarget"), ""},
		"uniqIds": []string{target.uniqId, notExistUniqId, target.uniqId},
	})

	target.expectCast(protocol.RouterSingleCastBulk, fromUserMessage{
		Message:  validData,
		FromUser: "uniqId(" + sender.uniqId + ")",
	})
	// 正常数据已收到：目标不存在的数据项被跳过，空数据项也不投递
	target.assertNoMessage(noMessageTimeout)
	other.assertNoMessage(noMessageTimeout)
}

// TestSingleCastBulkByCustomerId 按 customerId 批量单播：多条消息按顺序一一对应多个客户
func TestSingleCastBulkByCustomerId(t *testing.T) {
	sender := newWsClient(t)
	first := newWsClient(t)
	firstCustomerId := forgeSignIn(t, first)
	second := newWsClient(t)
	secondCustomerId := forgeSignIn(t, second)
	other := newWsClient(t)

	messageFirst := uniqueName("bulkByCustomerIdFirst")
	messageSecond := uniqueName("bulkByCustomerIdSecond")
	sender.send(protocol.RouterSingleCastBulkByCustomerId, map[string]any{
		"message":     []string{messageFirst, messageSecond},
		"customerIds": []string{firstCustomerId, secondCustomerId},
	})

	wantFrom := "uniqId(" + sender.uniqId + ")"
	first.expectCast(protocol.RouterSingleCastBulkByCustomerId, fromUserMessage{Message: messageFirst, FromUser: wantFrom})
	second.expectCast(protocol.RouterSingleCastBulkByCustomerId, fromUserMessage{Message: messageSecond, FromUser: wantFrom})
	other.assertNoMessage(noMessageTimeout)
}

// TestSingleCastBulkByCustomerIdSkipInvalidTarget 网关会跳过「customerId 不存在」与「数据为空」两类数据项
func TestSingleCastBulkByCustomerIdSkipInvalidTarget(t *testing.T) {
	sender := newWsClient(t)
	target := newWsClient(t)
	customerId := forgeSignIn(t, target)
	other := newWsClient(t)

	validData := uniqueName("bulkByCustomerIdValid")
	// 三个数据项依次对应：正常投递、customerId 不存在、数据为空
	sender.send(protocol.RouterSingleCastBulkByCustomerId, map[string]any{
		"message":     []string{validData, uniqueName("bulkByCustomerIdNotExist"), ""},
		"customerIds": []string{customerId, "不存在的customerId", customerId},
	})

	target.expectCast(protocol.RouterSingleCastBulkByCustomerId, fromUserMessage{
		Message:  validData,
		FromUser: "uniqId(" + sender.uniqId + ")",
	})
	// 正常数据已收到：customerId 不存在的数据项被跳过，空数据项也不投递
	target.assertNoMessage(noMessageTimeout)
	other.assertNoMessage(noMessageTimeout)
}

// TestBroadcast 广播：所有在线连接都能收到
func TestBroadcast(t *testing.T) {
	sender := newWsClient(t)
	other := newWsClient(t)

	message := uniqueName("broadcast")
	sender.send(protocol.RouterBroadcast, map[string]any{"message": message})

	want := fromUserMessage{Message: message, FromUser: "uniqId(" + sender.uniqId + ")"}
	sender.expectCast(protocol.RouterBroadcast, want)
	other.expectCast(protocol.RouterBroadcast, want)
}

// TestBroadcastBulk 批量广播：所有在线连接按顺序收到每一条数据，空消息对应的数据项被跳过
func TestBroadcastBulk(t *testing.T) {
	sender := newWsClient(t)
	first := newWsClient(t)
	second := newWsClient(t)

	messageFirst := uniqueName("broadcastBulkFirst")
	messageSecond := uniqueName("broadcastBulkSecond")
	sender.send(protocol.RouterBroadcastBulk, map[string]any{
		"message": []string{messageFirst, "", messageSecond},
	})

	fromUser := "uniqId(" + sender.uniqId + ")"
	for _, c := range []*wsClient{sender, first, second} {
		c.expectCast(protocol.RouterBroadcastBulk, fromUserMessage{Message: messageFirst, FromUser: fromUser})
		c.expectCast(protocol.RouterBroadcastBulk, fromUserMessage{Message: messageSecond, FromUser: fromUser})
		// 空消息对应的数据项被网关跳过，只应收到两条
		c.assertNoMessage(noMessageTimeout)
	}
}

// TestMulticast 组播：一条信息投递给多个 uniqId
func TestMulticast(t *testing.T) {
	sender := newWsClient(t)
	first := newWsClient(t)
	second := newWsClient(t)
	other := newWsClient(t)

	message := uniqueName("multicast")
	sender.send(protocol.RouterMulticast, map[string]any{
		"message": message,
		"unIqIds": []string{first.uniqId, second.uniqId},
	})

	want := fromUserMessage{Message: message, FromUser: "uniqId(" + sender.uniqId + ")"}
	first.expectCast(protocol.RouterMulticast, want)
	second.expectCast(protocol.RouterMulticast, want)
	other.assertNoMessage(noMessageTimeout)
}

// TestMulticastByCustomerId 组播：一条信息投递给多个 customerId
func TestMulticastByCustomerId(t *testing.T) {
	sender := newWsClient(t)
	first := newWsClient(t)
	firstCustomerId := forgeSignIn(t, first)
	second := newWsClient(t)
	secondCustomerId := forgeSignIn(t, second)
	other := newWsClient(t)

	message := uniqueName("multicastByCustomerId")
	sender.send(protocol.RouterMulticastByCustomerId, map[string]any{
		"message":     message,
		"customerIds": []string{firstCustomerId, secondCustomerId},
	})

	want := fromUserMessage{Message: message, FromUser: "uniqId(" + sender.uniqId + ")"}
	first.expectCast(protocol.RouterMulticastByCustomerId, want)
	second.expectCast(protocol.RouterMulticastByCustomerId, want)
	other.assertNoMessage(noMessageTimeout)
}
