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

package redisQueue

import (
	"context"
	"encoding/binary"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	internalMetrics "netsvr/internal/metrics"
	"testing"
	"time"
)

// TestLoopSendListBatchMode 测试loopSendList的批量发送分支（size < packLimit）
func TestLoopSendListBatchMode(t *testing.T) {
	// 记录初始metrics值
	initialCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	initialBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	redisAddr := "localhost:6379"
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_list_batch",
		KeyType: "list",
		DB:      0,
	}
	rdb.Del(ctx, queueConfig.Key)
	defer rdb.Del(ctx, queueConfig.Key)

	q := newQueue(rdb, queueConfig, 10)
	defer q.Close()

	msgCount := 10
	// 发送消息并保存原始数据用于验证
	type testData struct {
		msg      proto.Message
		cmd      netsvrProtocol.Cmd
		uniqId   string
		expected []byte // 期望的完整数据包（包含4字节cmd）
	}
	testDataList := make([]testData, msgCount)

	for i := 0; i < msgCount; i++ {
		msg := &netsvrProtocol.ConnOpen{
			UniqId:     string(rune('A' + i)),
			RemoteAddr: "127.0.0.1:8080",
		}
		testDataList[i] = testData{
			msg:    msg,
			cmd:    netsvrProtocol.Cmd_ConnOpen,
			uniqId: string(rune('A' + i)),
		}

		if size := q.Send(msg, netsvrProtocol.Cmd_ConnOpen); size <= 0 {
			t.Fatalf("第%d条消息发送失败", i)
		}
	}

	time.Sleep(300 * time.Millisecond)

	// 验证Redis数据
	if listLen := rdb.LLen(ctx, queueConfig.Key).Val(); int(listLen) != msgCount {
		t.Fatalf("期望Redis中有%d条数据，实际%d", msgCount, listLen)
	}

	// 从Redis读取数据并验证内容（注意：List是后进先出，所以反向遍历）
	for i := msgCount - 1; i >= 0; i-- {
		data, err := rdb.LPop(ctx, queueConfig.Key).Bytes()
		if err != nil {
			t.Fatalf("读取第%d条数据失败: %v", i, err)
		}
		if len(data) < 4 {
			t.Fatalf("第%d条数据长度不足4字节，实际长度: %d", i, len(data))
		}

		// 解析cmd
		cmd := binary.BigEndian.Uint32(data[0:4])
		if cmd != uint32(netsvrProtocol.Cmd_ConnOpen) {
			t.Fatalf("第%d条数据cmd错误，期望=%d, 实际=%d", i, netsvrProtocol.Cmd_ConnOpen, cmd)
		}

		// 反序列化body
		body := data[4:]
		receivedMsg := &netsvrProtocol.ConnOpen{}
		if err := proto.Unmarshal(body, receivedMsg); err != nil {
			t.Fatalf("第%d条数据反序列化失败: %v", i, err)
		}

		// 验证字段内容
		expectedUniqId := string(rune('A' + i))
		if receivedMsg.UniqId != expectedUniqId {
			t.Fatalf("第%d条数据UniqId不匹配，期望=%s, 实际=%s", i, expectedUniqId, receivedMsg.UniqId)
		}
		if receivedMsg.RemoteAddr != "127.0.0.1:8080" {
			t.Fatalf("第%d条数据RemoteAddr不匹配，期望=127.0.0.1:8080, 实际=%s", i, receivedMsg.RemoteAddr)
		}
	}

	// 验证metrics增量
	finalCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	finalBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	if deltaCount := finalCount - initialCount; deltaCount != int64(msgCount) {
		t.Fatalf("期望成功次数增量=%d, 实际=%d", msgCount, deltaCount)
	}
	if deltaBytes := finalBytes - initialBytes; deltaBytes <= 0 {
		t.Fatal("期望成功字节数增量>0")
	}

	t.Logf("List批量模式 - 次数增量:%d, 字节增量:%d", finalCount-initialCount, finalBytes-initialBytes)
}

// TestLoopSendListSingleMode 测试loopSendList的单个发送分支（size >= packLimit）
func TestLoopSendListSingleMode(t *testing.T) {
	initialCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	initialBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	redisAddr := "localhost:6379"
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_list_single",
		KeyType: "list",
		DB:      0,
	}
	rdb.Del(ctx, queueConfig.Key)
	defer rdb.Del(ctx, queueConfig.Key)

	q := newQueue(rdb, queueConfig, 2)
	defer q.Close()

	// dequeueSize=2，packLimit = max(2097152, 2*2*1024) = 2097152 (2MB)
	// 每条消息1.5MB，2条总共3MB > 2MB，确保走单个发送分支
	largeData := make([]byte, 1500*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	msgCount := 4
	// 发送消息并记录期望的数据
	type expectedData struct {
		uniqId     string
		customerId string
		data       []byte
	}
	expectedList := make([]expectedData, msgCount)

	for i := 0; i < msgCount; i++ {
		expectedList[i] = expectedData{
			uniqId:     string(rune('A' + i)),
			customerId: "test",
			data:       largeData,
		}
		msg := &netsvrProtocol.Transfer{
			UniqId:     expectedList[i].uniqId,
			CustomerId: expectedList[i].customerId,
			Data:       expectedList[i].data,
		}
		size := q.Send(msg, netsvrProtocol.Cmd_Transfer)
		if size <= 0 {
			t.Fatalf("第%d条大消息发送失败", i)
		}
		t.Logf("第%d条消息发送大小: %d字节 (%.2fKB)", i, size, float64(size)/1024)
	}

	t.Logf("总消息数: %d, dequeueSize: 2", msgCount)
	t.Logf("packLimit计算: max(2097152, 2*2*1024) = max(2097152, 4096) = 2097152 (2MB)")
	t.Logf("预期单次Dequeue: 2条 * 1.5MB = 3MB > 2MB，应该走单个发送分支")

	time.Sleep(300 * time.Millisecond)

	if listLen := rdb.LLen(ctx, queueConfig.Key).Val(); int(listLen) != msgCount {
		t.Fatalf("期望Redis中有%d条数据，实际%d", msgCount, listLen)
	}

	// 从Redis读取并验证每条数据的内容（注意：List是后进先出）
	for i := msgCount - 1; i >= 0; i-- {
		data, err := rdb.LPop(ctx, queueConfig.Key).Bytes()
		if err != nil {
			t.Fatalf("读取第%d条数据失败: %v", i, err)
		}
		if len(data) < 4 {
			t.Fatalf("第%d条数据长度不足4字节，实际长度: %d", i, len(data))
		}

		// 解析cmd
		cmd := binary.BigEndian.Uint32(data[0:4])
		if cmd != uint32(netsvrProtocol.Cmd_Transfer) {
			t.Fatalf("第%d条数据cmd错误，期望=%d, 实际=%d", i, netsvrProtocol.Cmd_Transfer, cmd)
		}

		// 反序列化body
		body := data[4:]
		receivedMsg := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(body, receivedMsg); err != nil {
			t.Fatalf("第%d条数据反序列化失败: %v", i, err)
		}

		// 验证字段内容
		if receivedMsg.UniqId != expectedList[i].uniqId {
			t.Fatalf("第%d条数据UniqId不匹配，期望=%s, 实际=%s", i, expectedList[i].uniqId, receivedMsg.UniqId)
		}
		if receivedMsg.CustomerId != expectedList[i].customerId {
			t.Fatalf("第%d条数据CustomerId不匹配，期望=%s, 实际=%s", i, expectedList[i].customerId, receivedMsg.CustomerId)
		}
		if len(receivedMsg.Data) != len(expectedList[i].data) {
			t.Fatalf("第%d条数据Data长度不匹配，期望=%d, 实际=%d", i, len(expectedList[i].data), len(receivedMsg.Data))
		}
		// 验证Data内容的每个字节
		for j := range receivedMsg.Data {
			if receivedMsg.Data[j] != expectedList[i].data[j] {
				t.Fatalf("第%d条数据Data[%d]不匹配，期望=%d, 实际=%d", i, j, expectedList[i].data[j], receivedMsg.Data[j])
			}
		}
	}

	finalCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	finalBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	if deltaCount := finalCount - initialCount; deltaCount != int64(msgCount) {
		t.Fatalf("期望成功次数增量=%d, 实际=%d", msgCount, deltaCount)
	}
	expectedMinBytes := int64(msgCount * (1500*1024 + 4))
	if deltaBytes := finalBytes - initialBytes; deltaBytes < expectedMinBytes {
		t.Fatalf("期望字节增量>=%d, 实际=%d", expectedMinBytes, deltaBytes)
	}

	t.Logf("List单个模式 - 次数增量:%d, 字节增量:%d", finalCount-initialCount, finalBytes-initialBytes)
}

// TestLoopSendStreamBatchMode 测试loopSendStream的批量发送分支
func TestLoopSendStreamBatchMode(t *testing.T) {
	initialCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	initialBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	redisAddr := "localhost:6379"
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_stream_batch",
		KeyType: "stream",
		DB:      0,
	}
	rdb.Del(ctx, queueConfig.Key)
	defer rdb.Del(ctx, queueConfig.Key)

	q := newQueue(rdb, queueConfig, 10)
	defer q.Close()

	msgCount := 10
	// 发送消息并记录期望的数据
	type expectedData struct {
		uniqId     string
		customerId string
	}
	expectedList := make([]expectedData, msgCount)

	for i := 0; i < msgCount; i++ {
		expectedList[i] = expectedData{
			uniqId:     string(rune('A' + i)),
			customerId: "batch",
		}
		msg := &netsvrProtocol.ConnClose{
			UniqId:     expectedList[i].uniqId,
			CustomerId: expectedList[i].customerId,
		}
		if size := q.Send(msg, netsvrProtocol.Cmd_ConnClose); size <= 0 {
			t.Fatalf("第%d条消息发送失败", i)
		}
	}

	time.Sleep(300 * time.Millisecond)

	if streamLen := rdb.XLen(ctx, queueConfig.Key).Val(); int(streamLen) != msgCount {
		t.Fatalf("期望Stream中有%d条数据，实际%d", msgCount, streamLen)
	}

	messages, err := rdb.XRead(ctx, &redis.XReadArgs{Streams: []string{queueConfig.Key, "0"}, Count: int64(msgCount)}).Result()
	if err != nil {
		t.Fatalf("读取Stream消息失败: %v", err)
	}
	if len(messages) == 0 || len(messages[0].Messages) != msgCount {
		t.Fatal("读取Stream消息数量不匹配")
	}

	// 验证每条消息的内容（Stream保持插入顺序）
	for i := 0; i < msgCount; i++ {
		msg := messages[0].Messages[i]
		data := msg.Values["data"]
		if data == nil {
			t.Fatalf("第%d条消息缺少data字段", i)
		}

		// 处理Redis返回的string或[]byte类型
		var dataBytes []byte
		switch v := data.(type) {
		case []byte:
			dataBytes = v
		case string:
			dataBytes = []byte(v)
		default:
			t.Fatalf("第%d条消息data类型错误: %T", i, data)
		}

		if len(dataBytes) < 4 {
			t.Fatalf("第%d条数据长度不足4字节，实际长度: %d", i, len(dataBytes))
		}

		// 解析cmd
		cmd := binary.BigEndian.Uint32(dataBytes[0:4])
		if cmd != uint32(netsvrProtocol.Cmd_ConnClose) {
			t.Fatalf("第%d条数据cmd错误，期望=%d, 实际=%d", i, netsvrProtocol.Cmd_ConnClose, cmd)
		}

		// 反序列化body
		body := dataBytes[4:]
		receivedMsg := &netsvrProtocol.ConnClose{}
		if err := proto.Unmarshal(body, receivedMsg); err != nil {
			t.Fatalf("第%d条数据反序列化失败: %v", i, err)
		}

		// 验证字段内容
		if receivedMsg.UniqId != expectedList[i].uniqId {
			t.Fatalf("第%d条数据UniqId不匹配，期望=%s, 实际=%s", i, expectedList[i].uniqId, receivedMsg.UniqId)
		}
		if receivedMsg.CustomerId != expectedList[i].customerId {
			t.Fatalf("第%d条数据CustomerId不匹配，期望=%s, 实际=%s", i, expectedList[i].customerId, receivedMsg.CustomerId)
		}
	}

	finalCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	finalBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	if deltaCount := finalCount - initialCount; deltaCount != int64(msgCount) {
		t.Fatalf("期望成功次数增量=%d, 实际=%d", msgCount, deltaCount)
	}
	if deltaBytes := finalBytes - initialBytes; deltaBytes <= 0 {
		t.Fatal("期望成功字节数增量>0")
	}

	t.Logf("Stream批量模式 - 次数增量:%d, 字节增量:%d", finalCount-initialCount, finalBytes-initialBytes)
}

// TestLoopSendStreamSingleMode 测试loopSendStream的单个发送分支
func TestLoopSendStreamSingleMode(t *testing.T) {
	initialCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	initialBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	redisAddr := "localhost:6379"
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_stream_single",
		KeyType: "stream",
		DB:      0,
	}
	rdb.Del(ctx, queueConfig.Key)
	defer rdb.Del(ctx, queueConfig.Key)

	q := newQueue(rdb, queueConfig, 2)
	defer q.Close()

	// dequeueSize=2，packLimit = max(2097152, 2*2*1024) = 2097152 (2MB)
	// 每条消息1.5MB，2条总共3MB > 2MB，确保走单个发送分支
	largeData := make([]byte, 1500*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	msgCount := 5
	// 发送消息并记录期望的数据
	type expectedData struct {
		uniqId     string
		customerId string
		data       []byte
	}
	expectedList := make([]expectedData, msgCount)

	for i := 0; i < msgCount; i++ {
		expectedList[i] = expectedData{
			uniqId:     string(rune('A' + i)),
			customerId: "single",
			data:       largeData,
		}
		msg := &netsvrProtocol.Transfer{
			UniqId:     expectedList[i].uniqId,
			CustomerId: expectedList[i].customerId,
			Data:       expectedList[i].data,
		}
		if size := q.Send(msg, netsvrProtocol.Cmd_Transfer); size <= 0 {
			t.Fatalf("第%d条大消息发送失败", i)
		}
	}

	time.Sleep(300 * time.Millisecond)

	if streamLen := rdb.XLen(ctx, queueConfig.Key).Val(); int(streamLen) != msgCount {
		t.Fatalf("期望Stream中有%d条数据，实际%d", msgCount, streamLen)
	}

	messages, _ := rdb.XRead(ctx, &redis.XReadArgs{Streams: []string{queueConfig.Key, "0"}, Count: int64(msgCount)}).Result()
	if len(messages) == 0 || len(messages[0].Messages) != msgCount {
		t.Fatal("读取Stream消息失败")
	}

	// 验证每条消息的内容（Stream保持插入顺序）
	for i := 0; i < msgCount; i++ {
		msg := messages[0].Messages[i]
		data := msg.Values["data"]
		if data == nil {
			t.Fatalf("第%d条消息缺少data字段", i)
		}
		var dataBytes []byte
		switch v := data.(type) {
		case []byte:
			dataBytes = v
		case string:
			dataBytes = []byte(v)
		default:
			t.Fatalf("第%d条消息data类型错误: %T", i, data)
		}

		if len(dataBytes) < 4 {
			t.Fatalf("第%d条数据长度不足4字节，实际长度: %d", i, len(dataBytes))
		}

		// 解析cmd
		cmd := binary.BigEndian.Uint32(dataBytes[0:4])
		if cmd != uint32(netsvrProtocol.Cmd_Transfer) {
			t.Fatalf("第%d条数据cmd错误，期望=%d, 实际=%d", i, netsvrProtocol.Cmd_Transfer, cmd)
		}

		// 反序列化body
		body := dataBytes[4:]
		receivedMsg := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(body, receivedMsg); err != nil {
			t.Fatalf("第%d条数据反序列化失败: %v", i, err)
		}

		// 验证字段内容
		if receivedMsg.UniqId != expectedList[i].uniqId {
			t.Fatalf("第%d条数据UniqId不匹配，期望=%s, 实际=%s", i, expectedList[i].uniqId, receivedMsg.UniqId)
		}
		if receivedMsg.CustomerId != expectedList[i].customerId {
			t.Fatalf("第%d条数据CustomerId不匹配，期望=%s, 实际=%s", i, expectedList[i].customerId, receivedMsg.CustomerId)
		}
		if len(receivedMsg.Data) != len(expectedList[i].data) {
			t.Fatalf("第%d条数据Data长度不匹配，期望=%d, 实际=%d", i, len(expectedList[i].data), len(receivedMsg.Data))
		}
		// 验证Data内容的每个字节
		for j := range receivedMsg.Data {
			if receivedMsg.Data[j] != expectedList[i].data[j] {
				t.Fatalf("第%d条数据Data[%d]不匹配，期望=%d, 实际=%d", i, j, expectedList[i].data[j], receivedMsg.Data[j])
			}
		}
	}

	finalCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	finalBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	if deltaCount := finalCount - initialCount; deltaCount != int64(msgCount) {
		t.Fatalf("期望成功次数增量=%d, 实际=%d", msgCount, deltaCount)
	}
	expectedMinBytes := int64(msgCount * (1500*1024 + 4))
	if deltaBytes := finalBytes - initialBytes; deltaBytes < expectedMinBytes {
		t.Fatalf("期望字节增量>=%d, 实际=%d", expectedMinBytes, deltaBytes)
	}

	t.Logf("Stream单个模式 - 次数增量:%d, 字节增量:%d", finalCount-initialCount, finalBytes-initialBytes)
}

// TestLoopSendListQueueClose 测试loopSendList退出逻辑
func TestLoopSendListQueueClose(t *testing.T) {
	redisAddr := "localhost:6379"
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_list_close",
		KeyType: "list",
		DB:      0,
	}
	rdb.Del(ctx, queueConfig.Key)
	defer rdb.Del(ctx, queueConfig.Key)

	q := newQueue(rdb, queueConfig, 10)

	for i := 0; i < 5; i++ {
		q.Send(&netsvrProtocol.ConnOpen{UniqId: string(rune('A' + i))}, netsvrProtocol.Cmd_ConnOpen)
	}

	q.Close()
	time.Sleep(300 * time.Millisecond)

	if !q.sendCh.IsClosed() {
		t.Fatal("期望sendCh已关闭")
	}
}

// TestLoopSendStreamQueueClose 测试loopSendStream退出逻辑
func TestLoopSendStreamQueueClose(t *testing.T) {
	redisAddr := "localhost:6379"
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_stream_close",
		KeyType: "stream",
		DB:      0,
	}
	rdb.Del(ctx, queueConfig.Key)
	defer rdb.Del(ctx, queueConfig.Key)

	q := newQueue(rdb, queueConfig, 10)

	for i := 0; i < 5; i++ {
		q.Send(&netsvrProtocol.ConnClose{UniqId: string(rune('A' + i))}, netsvrProtocol.Cmd_ConnClose)
	}

	q.Close()
	time.Sleep(300 * time.Millisecond)

	if !q.sendCh.IsClosed() {
		t.Fatal("期望sendCh已关闭")
	}
}

// TestLoopSendMixedSizes 测试混合大小消息
func TestLoopSendMixedSizes(t *testing.T) {
	initialCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	initialBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	redisAddr := "localhost:6379"
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_mixed",
		KeyType: "list",
		DB:      0,
	}
	rdb.Del(ctx, queueConfig.Key)
	defer rdb.Del(ctx, queueConfig.Key)

	q := newQueue(rdb, queueConfig, 8)
	defer q.Close()

	// 发送小消息并记录期望数据
	smallMsgCount := 8
	type expectedSmallData struct {
		uniqId string
	}
	smallExpectedList := make([]expectedSmallData, smallMsgCount)

	for i := 0; i < smallMsgCount; i++ {
		smallExpectedList[i] = expectedSmallData{
			uniqId: string(rune('a' + i)),
		}
		q.Send(&netsvrProtocol.ConnOpen{UniqId: smallExpectedList[i].uniqId}, netsvrProtocol.Cmd_ConnOpen)
	}
	time.Sleep(200 * time.Millisecond)

	// 发送大消息并记录期望数据
	// 每条消息500KB，5条总共2.5MB，确保超过packLimit(2MB)
	largeData := make([]byte, 500*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	largeMsgCount := 5
	type expectedLargeData struct {
		uniqId string
		data   []byte
	}
	largeExpectedList := make([]expectedLargeData, largeMsgCount)

	for i := 0; i < largeMsgCount; i++ {
		largeExpectedList[i] = expectedLargeData{
			uniqId: string(rune('A' + i)),
			data:   largeData,
		}
		q.Send(&netsvrProtocol.Transfer{UniqId: largeExpectedList[i].uniqId, Data: largeExpectedList[i].data}, netsvrProtocol.Cmd_Transfer)
	}
	time.Sleep(200 * time.Millisecond)

	totalExpected := smallMsgCount + largeMsgCount
	if listLen := rdb.LLen(ctx, queueConfig.Key).Val(); int(listLen) != totalExpected {
		t.Fatalf("期望Redis中有%d条数据，实际%d", totalExpected, listLen)
	}

	// 从Redis读取数据并验证（注意：List是后进先出，所以先读大消息，再读小消息）
	// 先验证大消息（后入的，先出）
	for i := largeMsgCount - 1; i >= 0; i-- {
		data, err := rdb.LPop(ctx, queueConfig.Key).Bytes()
		if err != nil {
			t.Fatalf("读取第%d条大消息失败: %v", i, err)
		}
		if len(data) < 4 {
			t.Fatalf("第%d条大消息长度不足4字节，实际长度: %d", i, len(data))
		}

		// 解析cmd
		cmd := binary.BigEndian.Uint32(data[0:4])
		if cmd != uint32(netsvrProtocol.Cmd_Transfer) {
			t.Fatalf("第%d条大消息cmd错误，期望=%d, 实际=%d", i, netsvrProtocol.Cmd_Transfer, cmd)
		}

		// 反序列化body
		body := data[4:]
		receivedMsg := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(body, receivedMsg); err != nil {
			t.Fatalf("第%d条大消息反序列化失败: %v", i, err)
		}

		// 验证字段内容
		if receivedMsg.UniqId != largeExpectedList[i].uniqId {
			t.Fatalf("第%d条大消息UniqId不匹配，期望=%s, 实际=%s", i, largeExpectedList[i].uniqId, receivedMsg.UniqId)
		}
		if len(receivedMsg.Data) != len(largeExpectedList[i].data) {
			t.Fatalf("第%d条大消息Data长度不匹配，期望=%d, 实际=%d", i, len(largeExpectedList[i].data), len(receivedMsg.Data))
		}
		// 验证Data内容的每个字节
		for j := range receivedMsg.Data {
			if receivedMsg.Data[j] != largeExpectedList[i].data[j] {
				t.Fatalf("第%d条大消息Data[%d]不匹配，期望=%d, 实际=%d", i, j, largeExpectedList[i].data[j], receivedMsg.Data[j])
			}
		}
	}

	// 再验证小消息
	for i := smallMsgCount - 1; i >= 0; i-- {
		data, err := rdb.LPop(ctx, queueConfig.Key).Bytes()
		if err != nil {
			t.Fatalf("读取第%d条小消息失败: %v", i, err)
		}
		if len(data) < 4 {
			t.Fatalf("第%d条小消息长度不足4字节，实际长度: %d", i, len(data))
		}

		// 解析cmd
		cmd := binary.BigEndian.Uint32(data[0:4])
		if cmd != uint32(netsvrProtocol.Cmd_ConnOpen) {
			t.Fatalf("第%d条小消息cmd错误，期望=%d, 实际=%d", i, netsvrProtocol.Cmd_ConnOpen, cmd)
		}

		// 反序列化body
		body := data[4:]
		receivedMsg := &netsvrProtocol.ConnOpen{}
		if err := proto.Unmarshal(body, receivedMsg); err != nil {
			t.Fatalf("第%d条小消息反序列化失败: %v", i, err)
		}

		// 验证字段内容
		if receivedMsg.UniqId != smallExpectedList[i].uniqId {
			t.Fatalf("第%d条小消息UniqId不匹配，期望=%s, 实际=%s", i, smallExpectedList[i].uniqId, receivedMsg.UniqId)
		}
	}

	finalCount := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedCount].Meter.Snapshot().Count()
	finalBytes := internalMetrics.Registry[internalMetrics.ItemRedisQueueToBusinessSucceedByte].Meter.Snapshot().Count()

	if deltaCount := finalCount - initialCount; deltaCount != int64(totalExpected) {
		t.Fatalf("期望成功次数增量=%d, 实际=%d", totalExpected, deltaCount)
	}
	if deltaBytes := finalBytes - initialBytes; deltaBytes <= 0 {
		t.Fatal("期望成功字节数增量>0")
	}

	t.Logf("混合模式 - 次数增量:%d, 字节增量:%d (小:%d, 大:%d)", finalCount-initialCount, finalBytes-initialBytes, smallMsgCount, largeMsgCount)
}
