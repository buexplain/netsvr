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
	"encoding/binary"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
	"netsvr/configs"
	internalMetrics "netsvr/internal/metrics"
	"netsvr/pkg/quit"
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

	if _, err := rdb.Ping(quit.Ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_list_batch",
		KeyType: "list",
		DB:      new(int),
	}
	rdb.Del(quit.Ctx, queueConfig.Key)
	defer rdb.Del(quit.Ctx, queueConfig.Key)

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

		q.Send(msg, netsvrProtocol.Cmd_ConnOpen)
	}

	time.Sleep(300 * time.Millisecond)

	// 验证Redis数据
	if listLen := rdb.LLen(quit.Ctx, queueConfig.Key).Val(); int(listLen) != msgCount {
		t.Fatalf("期望Redis中有%d条数据，实际%d", msgCount, listLen)
	}

	// 从Redis读取数据并验证内容（注意：List是后进先出，但协程池写入是乱序的，需要基于UniqId验证）
	// 读取所有数据到map中，通过UniqId进行验证
	receivedMap := make(map[string]*netsvrProtocol.ConnOpen)
	for i := 0; i < msgCount; i++ {
		data, err := rdb.LPop(quit.Ctx, queueConfig.Key).Bytes()
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

		// 存入map，以UniqId为key
		receivedMap[receivedMsg.UniqId] = receivedMsg
	}

	// 基于UniqId验证每条消息
	for i := 0; i < msgCount; i++ {
		expectedUniqId := string(rune('A' + i))
		receivedMsg, exists := receivedMap[expectedUniqId]
		if !exists {
			t.Fatalf("未找到UniqId=%s的消息", expectedUniqId)
		}
		if receivedMsg.RemoteAddr != "127.0.0.1:8080" {
			t.Fatalf("UniqId=%s的消息RemoteAddr不匹配，期望=127.0.0.1:8080, 实际=%s", expectedUniqId, receivedMsg.RemoteAddr)
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

	if _, err := rdb.Ping(quit.Ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_list_single",
		KeyType: "list",
		DB:      new(int),
	}
	rdb.Del(quit.Ctx, queueConfig.Key)
	defer rdb.Del(quit.Ctx, queueConfig.Key)

	q := newQueue(rdb, queueConfig, 2)
	defer q.Close()

	// dequeueSize=2，packLimit = max(2097152, 2*2*1024) = 2097152 (2MB)
	// 每条消息3MB，确保走单个发送分支
	largeData := make([]byte, 3*1024*1024)
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
		q.Send(msg, netsvrProtocol.Cmd_Transfer)
	}

	t.Logf("总消息数: %d, dequeueSize: 2", msgCount)
	t.Logf("packLimit计算: max(2097152, 2*2*1024) = max(2097152, 4096) = 2097152 (2MB)")
	t.Logf("预期单次Dequeue: 2条 * 1.5MB = 3MB > 2MB，应该走单个发送分支")

	time.Sleep(1000 * time.Millisecond)

	if listLen := rdb.LLen(quit.Ctx, queueConfig.Key).Val(); int(listLen) != msgCount {
		t.Fatalf("期望Redis中有%d条数据，实际%d", msgCount, listLen)
	}

	// 从Redis读取并验证每条数据的内容（注意：List是后进先出，但协程池写入是乱序的，需要基于UniqId验证）
	// 读取所有数据到map中，通过UniqId进行验证
	type receivedTransferData struct {
		msg      *netsvrProtocol.Transfer
		expected expectedData
	}
	receivedTransferMap := make(map[string]*receivedTransferData)

	for i := 0; i < msgCount; i++ {
		data, err := rdb.LPop(quit.Ctx, queueConfig.Key).Bytes()
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

		// 存入map，以UniqId为key
		receivedTransferMap[receivedMsg.UniqId] = &receivedTransferData{
			msg:      receivedMsg,
			expected: expectedList[i],
		}
	}

	// 基于UniqId验证每条消息
	for i := 0; i < msgCount; i++ {
		expectedUniqId := expectedList[i].uniqId
		receivedData, exists := receivedTransferMap[expectedUniqId]
		if !exists {
			t.Fatalf("未找到UniqId=%s的消息", expectedUniqId)
		}
		receivedMsg := receivedData.msg
		expected := receivedData.expected

		if receivedMsg.CustomerId != expected.customerId {
			t.Fatalf("UniqId=%s的消息CustomerId不匹配，期望=%s, 实际=%s", expectedUniqId, expected.customerId, receivedMsg.CustomerId)
		}
		if len(receivedMsg.Data) != len(expected.data) {
			t.Fatalf("UniqId=%s的消息Data长度不匹配，期望=%d, 实际=%d", expectedUniqId, len(expected.data), len(receivedMsg.Data))
		}
		// 验证Data内容的每个字节
		for j := range receivedMsg.Data {
			if receivedMsg.Data[j] != expected.data[j] {
				t.Fatalf("UniqId=%s的消息Data[%d]不匹配，期望=%d, 实际=%d", expectedUniqId, j, expected.data[j], receivedMsg.Data[j])
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

	if _, err := rdb.Ping(quit.Ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_stream_batch",
		KeyType: "stream",
		DB:      new(int),
	}
	rdb.Del(quit.Ctx, queueConfig.Key)
	defer rdb.Del(quit.Ctx, queueConfig.Key)

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
		q.Send(msg, netsvrProtocol.Cmd_ConnClose)
	}

	time.Sleep(300 * time.Millisecond)

	if streamLen := rdb.XLen(quit.Ctx, queueConfig.Key).Val(); int(streamLen) != msgCount {
		t.Fatalf("期望Stream中有%d条数据，实际%d", msgCount, streamLen)
	}

	messages, err := rdb.XRead(quit.Ctx, &redis.XReadArgs{Streams: []string{queueConfig.Key, "0"}, Count: int64(msgCount)}).Result()
	if err != nil {
		t.Fatalf("读取Stream消息失败: %v", err)
	}
	if len(messages) == 0 || len(messages[0].Messages) != msgCount {
		t.Fatal("读取Stream消息数量不匹配")
	}

	// 验证每条消息的内容（Stream保持插入顺序，但协程池写入是乱序的，需要基于UniqId验证）
	// 读取所有数据到map中，通过UniqId进行验证
	receivedConnCloseMap := make(map[string]*netsvrProtocol.ConnClose)

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

		// 存入map，以UniqId为key
		receivedConnCloseMap[receivedMsg.UniqId] = receivedMsg
	}

	// 基于UniqId验证每条消息
	for i := 0; i < msgCount; i++ {
		expectedUniqId := expectedList[i].uniqId
		receivedMsg, exists := receivedConnCloseMap[expectedUniqId]
		if !exists {
			t.Fatalf("未找到UniqId=%s的消息", expectedUniqId)
		}
		if receivedMsg.CustomerId != expectedList[i].customerId {
			t.Fatalf("UniqId=%s的消息CustomerId不匹配，期望=%s, 实际=%s", expectedUniqId, expectedList[i].customerId, receivedMsg.CustomerId)
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

	if _, err := rdb.Ping(quit.Ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_stream_single",
		KeyType: "stream",
		DB:      new(int),
	}
	rdb.Del(quit.Ctx, queueConfig.Key)
	defer rdb.Del(quit.Ctx, queueConfig.Key)

	q := newQueue(rdb, queueConfig, 2)
	defer q.Close()

	// dequeueSize=2，packLimit = max(2097152, 2*2*1024) = 2097152 (2MB)
	// 每条消息3MB，确保走单个发送分支
	largeData := make([]byte, 3*1024*1024)
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
		q.Send(msg, netsvrProtocol.Cmd_Transfer)
	}

	time.Sleep(1000 * time.Millisecond)

	if streamLen := rdb.XLen(quit.Ctx, queueConfig.Key).Val(); int(streamLen) != msgCount {
		t.Fatalf("期望Stream中有%d条数据，实际%d", msgCount, streamLen)
	}

	messages, _ := rdb.XRead(quit.Ctx, &redis.XReadArgs{Streams: []string{queueConfig.Key, "0"}, Count: int64(msgCount)}).Result()
	if len(messages) == 0 || len(messages[0].Messages) != msgCount {
		t.Fatal("读取Stream消息失败")
	}

	// 验证每条消息的内容（Stream保持插入顺序，但协程池写入是乱序的，需要基于UniqId验证）
	// 读取所有数据到map中，通过UniqId进行验证
	receivedTransferStreamMap := make(map[string]*netsvrProtocol.Transfer)

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

		// 存入map，以UniqId为key
		receivedTransferStreamMap[receivedMsg.UniqId] = receivedMsg
	}

	// 基于UniqId验证每条消息
	for i := 0; i < msgCount; i++ {
		expectedUniqId := expectedList[i].uniqId
		receivedMsg, exists := receivedTransferStreamMap[expectedUniqId]
		if !exists {
			t.Fatalf("未找到UniqId=%s的消息", expectedUniqId)
		}
		expected := expectedList[i]

		if receivedMsg.CustomerId != expected.customerId {
			t.Fatalf("UniqId=%s的消息CustomerId不匹配，期望=%s, 实际=%s", expectedUniqId, expected.customerId, receivedMsg.CustomerId)
		}
		if len(receivedMsg.Data) != len(expected.data) {
			t.Fatalf("UniqId=%s的消息Data长度不匹配，期望=%d, 实际=%d", expectedUniqId, len(expected.data), len(receivedMsg.Data))
		}
		// 验证Data内容的每个字节
		for j := range receivedMsg.Data {
			if receivedMsg.Data[j] != expected.data[j] {
				t.Fatalf("UniqId=%s的消息Data[%d]不匹配，期望=%d, 实际=%d", expectedUniqId, j, expected.data[j], receivedMsg.Data[j])
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

	if _, err := rdb.Ping(quit.Ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_list_close",
		KeyType: "list",
		DB:      new(int),
	}
	rdb.Del(quit.Ctx, queueConfig.Key)
	defer rdb.Del(quit.Ctx, queueConfig.Key)

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

	if _, err := rdb.Ping(quit.Ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_stream_close",
		KeyType: "stream",
		DB:      new(int),
	}
	rdb.Del(quit.Ctx, queueConfig.Key)
	defer rdb.Del(quit.Ctx, queueConfig.Key)

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

	if _, err := rdb.Ping(quit.Ctx).Result(); err != nil {
		t.Fatalf("Redis不可用: %v", err)
	}

	queueConfig := configs.RedisQueue{
		Address: redisAddr,
		Key:     "test_mixed",
		KeyType: "list",
		DB:      new(int),
	}
	rdb.Del(quit.Ctx, queueConfig.Key)
	defer rdb.Del(quit.Ctx, queueConfig.Key)

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
	time.Sleep(500 * time.Millisecond)

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
	time.Sleep(500 * time.Millisecond)

	totalExpected := smallMsgCount + largeMsgCount
	if listLen := rdb.LLen(quit.Ctx, queueConfig.Key).Val(); int(listLen) != totalExpected {
		t.Fatalf("期望Redis中有%d条数据，实际%d", totalExpected, listLen)
	}

	// 从Redis读取数据并验证（注意：List是后进先出，但协程池写入是乱序的，需要基于UniqId验证）
	// 读取所有数据到map中，通过UniqId和cmd类型进行验证
	type mixedReceivedData struct {
		cmdType  string // "small" or "large"
		connOpen *netsvrProtocol.ConnOpen
		transfer *netsvrProtocol.Transfer
	}
	mixedReceivedMap := make(map[string]*mixedReceivedData)

	totalExpected = smallMsgCount + largeMsgCount
	for i := 0; i < totalExpected; i++ {
		data, err := rdb.LPop(quit.Ctx, queueConfig.Key).Bytes()
		if err != nil {
			t.Fatalf("读取第%d条数据失败: %v", i, err)
		}
		if len(data) < 4 {
			t.Fatalf("第%d条数据长度不足4字节，实际长度: %d", i, len(data))
		}

		// 解析cmd
		cmd := binary.BigEndian.Uint32(data[0:4])
		body := data[4:]

		if cmd == uint32(netsvrProtocol.Cmd_Transfer) {
			// 大消息
			receivedMsg := &netsvrProtocol.Transfer{}
			if err := proto.Unmarshal(body, receivedMsg); err != nil {
				t.Fatalf("第%d条大消息反序列化失败: %v", i, err)
			}
			mixedReceivedMap[receivedMsg.UniqId] = &mixedReceivedData{
				cmdType:  "large",
				transfer: receivedMsg,
			}
		} else if cmd == uint32(netsvrProtocol.Cmd_ConnOpen) {
			// 小消息
			receivedMsg := &netsvrProtocol.ConnOpen{}
			if err := proto.Unmarshal(body, receivedMsg); err != nil {
				t.Fatalf("第%d条小消息反序列化失败: %v", i, err)
			}
			mixedReceivedMap[receivedMsg.UniqId] = &mixedReceivedData{
				cmdType:  "small",
				connOpen: receivedMsg,
			}
		} else {
			t.Fatalf("第%d条数据cmd错误，实际=%d", i, cmd)
		}
	}

	// 基于UniqId验证大消息
	for i := 0; i < largeMsgCount; i++ {
		expectedUniqId := largeExpectedList[i].uniqId
		receivedData, exists := mixedReceivedMap[expectedUniqId]
		if !exists {
			t.Fatalf("未找到UniqId=%s的大消息", expectedUniqId)
		}
		if receivedData.cmdType != "large" {
			t.Fatalf("UniqId=%s的消息类型错误，期望=large, 实际=%s", expectedUniqId, receivedData.cmdType)
		}
		receivedMsg := receivedData.transfer
		expected := largeExpectedList[i]

		if len(receivedMsg.Data) != len(expected.data) {
			t.Fatalf("UniqId=%s的大消息Data长度不匹配，期望=%d, 实际=%d", expectedUniqId, len(expected.data), len(receivedMsg.Data))
		}
		// 验证Data内容的每个字节
		for j := range receivedMsg.Data {
			if receivedMsg.Data[j] != expected.data[j] {
				t.Fatalf("UniqId=%s的大消息Data[%d]不匹配，期望=%d, 实际=%d", expectedUniqId, j, expected.data[j], receivedMsg.Data[j])
			}
		}
	}

	// 基于UniqId验证小消息
	for i := 0; i < smallMsgCount; i++ {
		expectedUniqId := smallExpectedList[i].uniqId
		receivedData, exists := mixedReceivedMap[expectedUniqId]
		if !exists {
			t.Fatalf("未找到UniqId=%s的小消息", expectedUniqId)
		}
		if receivedData.cmdType != "small" {
			t.Fatalf("UniqId=%s的消息类型错误，期望=small, 实际=%s", expectedUniqId, receivedData.cmdType)
		}
		receivedMsg := receivedData.connOpen

		if receivedMsg.UniqId != expectedUniqId {
			t.Fatalf("小消息UniqId不匹配，期望=%s, 实际=%s", expectedUniqId, receivedMsg.UniqId)
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
