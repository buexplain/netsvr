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

// 本文件覆盖主题相关命令：订阅、退订、删除、发布、批量发布，以及主题维度的各类查询。
package integration

import (
	"slices"
	"testing"

	"netsvr/test/pkg/protocol"
)

// TestTopicSubscribeAndPublish 订阅后主题会出现在网关的主题列表里，发布的内容只有订阅者能收到
func TestTopicSubscribeAndPublish(t *testing.T) {
	subscriber := newWsClient(t)
	anotherSubscriber := newWsClient(t)
	outsider := newWsClient(t)
	topic := uniqueName("topic")

	subscriber.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{topic}})
	anotherSubscriber.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{topic}})
	subscribers := uniqIds(subscriber, anotherSubscriber)

	env := outsider.callOK(protocol.RouterTopicList, nil)
	var topicList topicListPayload
	env.into(t, &topicList)
	assertContains(t, "topicList", []string{topic}, topicList.all())

	env = outsider.callOK(protocol.RouterTopicCount, nil)
	var topicCount countPayload
	env.into(t, &topicCount)
	if int(topicCount.Count) != len(topicList.all()) {
		t.Fatalf("topicCount 与 topicList 不一致：count=%d，list=%d", topicCount.Count, len(topicList.all()))
	}

	env = outsider.callOK(protocol.RouterTopicUniqIdList, map[string]any{"topics": []string{topic}})
	var uniqIdList topicUniqIdListPayload
	env.into(t, &uniqIdList)
	assertSameSet(t, "主题下的uniqId", subscribers, uniqIdList.topicUniqIds(topic))

	env = outsider.callOK(protocol.RouterTopicUniqIdCount, map[string]any{"topics": []string{topic}})
	var uniqIdCount topicUniqIdCountPayload
	env.into(t, &uniqIdCount)
	if got := uniqIdCount.topicCount(topic); got != 2 {
		t.Fatalf("主题下的连接数不符合预期：期望 2，实际 %d", got)
	}

	message := uniqueName("topicPublish")
	subscriber.send(protocol.RouterTopicPublish, map[string]any{"message": message, "topics": []string{topic}})
	want := fromUserMessage{Message: message, FromUser: "uniqId(" + subscriber.uniqId + ")"}
	subscriber.expectCast(protocol.RouterTopicPublish, want)
	anotherSubscriber.expectCast(protocol.RouterTopicPublish, want)
	outsider.assertNoMessage(noMessageTimeout)
}

// TestTopicUnsubscribe 取消订阅后不再收到该主题的发布，主题下的连接列表也会同步变化
func TestTopicUnsubscribe(t *testing.T) {
	subscriber := newWsClient(t)
	other := newWsClient(t)
	publisher := newWsClient(t)
	topic := uniqueName("topic")

	subscriber.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{topic}})
	other.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{topic}})
	subscriber.callOK(protocol.RouterTopicUnsubscribe, map[string]any{"topics": []string{topic}})

	env := other.callOK(protocol.RouterTopicUniqIdList, map[string]any{"topics": []string{topic}})
	var uniqIdList topicUniqIdListPayload
	env.into(t, &uniqIdList)
	assertSameSet(t, "退订后主题下的uniqId", []string{other.uniqId}, uniqIdList.topicUniqIds(topic))

	publisher.send(protocol.RouterTopicPublish, map[string]any{"message": uniqueName("publish"), "topics": []string{topic}})
	if cmd, env := other.read(); cmd != protocol.RouterTopicPublish {
		t.Fatalf("订阅者应收到发布消息，实际 cmd=%d", cmd)
	} else {
		requireOK(t, env)
	}
	subscriber.assertNoMessage(noMessageTimeout)
}

// TestTopicDelete 删除主题会移除网关中的主题，并给每个曾订阅该主题的连接各发一次通知
func TestTopicDelete(t *testing.T) {
	first := newWsClient(t)
	second := newWsClient(t)
	outsider := newWsClient(t)
	topic := uniqueName("topic")

	first.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{topic}})
	second.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{topic}})

	first.send(protocol.RouterTopicDelete, map[string]any{"topics": []string{topic}})
	requireOK(t, first.waitFor(protocol.RouterTopicDelete))
	requireOK(t, second.waitFor(protocol.RouterTopicDelete))
	outsider.assertNoMessage(noMessageTimeout)

	// 主题已从网关移除
	env := outsider.callOK(protocol.RouterTopicList, nil)
	var topicList topicListPayload
	env.into(t, &topicList)
	if slices.Contains(topicList.all(), topic) {
		t.Fatalf("主题 %s 删除后不应再出现在主题列表中：%v", topic, topicList.all())
	}

	// 连接上存储的主题也被清除
	env = first.callOK(protocol.RouterConnInfo, nil)
	var info connInfoPayload
	env.into(t, &info)
	item, ok := info.items()[first.uniqId]
	if !ok {
		t.Fatalf("删除主题后应能查到自己的连接信息")
	}
	if slices.Contains(item.Topics, topic) {
		t.Fatalf("主题 %s 删除后不应还挂在连接上：%v", topic, item.Topics)
	}
}

// TestTopicCustomerIdQueries 主题的 customerId 维度的列表、数量与映射关系
func TestTopicCustomerIdQueries(t *testing.T) {
	first := newWsClient(t)
	firstCustomerId := forgeSignIn(t, first)
	second := newWsClient(t)
	secondCustomerId := forgeSignIn(t, second)
	observer := newWsClient(t)
	topic := uniqueName("topic")

	first.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{topic}})
	second.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{topic}})

	env := observer.callOK(protocol.RouterTopicCustomerIdList, map[string]any{"topics": []string{topic}})
	var customerIdList topicCustomerIdListPayload
	env.into(t, &customerIdList)
	assertSameSet(t, "主题下的customerId", []string{firstCustomerId, secondCustomerId}, customerIdList.topicCustomerIds(topic))

	env = observer.callOK(protocol.RouterTopicCustomerIdCount, map[string]any{"topics": []string{topic}})
	var customerIdCount topicCustomerIdCountPayload
	env.into(t, &customerIdCount)
	if got := customerIdCount.topicCount(topic); got != 2 {
		t.Fatalf("主题下的customerId数量不符合预期：期望 2，实际 %d", got)
	}

	env = observer.callOK(protocol.RouterTopicCustomerIdToUniqIdsList, map[string]any{"topics": []string{topic}})
	var mapping topicCustomerIdToUniqIdsListPayload
	env.into(t, &mapping)
	assertSameSet(t, "customerId 对应的 uniqId", []string{first.uniqId}, mapping.customerUniqIds(topic, firstCustomerId))
	assertSameSet(t, "customerId 对应的 uniqId", []string{second.uniqId}, mapping.customerUniqIds(topic, secondCustomerId))

	// 不存在的主题不应返回任何数据
	if got := customerIdList.topicCustomerIds("不存在的主题"); len(got) != 0 {
		t.Fatalf("不存在的主题不应返回 customerId，实际 %v", got)
	}
}

// TestTopicPublishBulk 批量发布：多条消息与多个主题按顺序一一对应，空消息对应的数据项被跳过
func TestTopicPublishBulk(t *testing.T) {
	firstTopic := uniqueName("topic")
	secondTopic := uniqueName("topic")
	emptyTopic := uniqueName("topic")
	firstSubscriber := newWsClient(t)
	secondSubscriber := newWsClient(t)
	thirdSubscriber := newWsClient(t)
	fourthSubscriber := newWsClient(t)
	emptySubscriber := newWsClient(t)

	firstSubscriber.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{firstTopic}})
	secondSubscriber.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{firstTopic}})
	thirdSubscriber.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{secondTopic}})
	fourthSubscriber.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{secondTopic}})
	emptySubscriber.callOK(protocol.RouterTopicSubscribe, map[string]any{"topics": []string{emptyTopic}})

	sender := newWsClient(t)
	firstMessage := uniqueName("bulkTopicFirst")
	secondMessage := uniqueName("bulkTopicSecond")
	sender.send(protocol.RouterTopicPublishBulk, map[string]any{
		"message": []string{firstMessage, secondMessage, ""},
		"topics":  []string{firstTopic, secondTopic, emptyTopic},
	})

	fromUser := "uniqId(" + sender.uniqId + ")"
	wantFirst := fromUserMessage{Message: firstMessage, FromUser: fromUser}
	wantSecond := fromUserMessage{Message: secondMessage, FromUser: fromUser}
	firstSubscriber.expectCast(protocol.RouterTopicPublishBulk, wantFirst)
	secondSubscriber.expectCast(protocol.RouterTopicPublishBulk, wantFirst)
	thirdSubscriber.expectCast(protocol.RouterTopicPublishBulk, wantSecond)
	fourthSubscriber.expectCast(protocol.RouterTopicPublishBulk, wantSecond)
	// 空消息对应的数据项被网关跳过，该主题的订阅者收不到任何数据
	emptySubscriber.assertNoMessage(noMessageTimeout)
}
