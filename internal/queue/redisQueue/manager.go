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

// Package redisQueue Redis队列
package redisQueue

import (
	"context"
	"fmt"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/redis/go-redis/v9"
	"netsvr/configs"
	"netsvr/internal/log"
	"os"
	"time"
)

// 数组大小基于协议中最大的 Event 枚举值
const managerLen = netsvrProtocol.Event_OnMessage + 1

type manager [managerLen]*Queue

func (r manager) Get(event netsvrProtocol.Event) *Queue {
	return r[event]
}

// Manager 管理所有的redis队列
var Manager manager

func init() {
	// 验证协议中的 Event 枚举值是否超出数组范围
	maxUsedEvent := 0
	for _, v := range netsvrProtocol.Event_value {
		maxUsedEvent = max(maxUsedEvent, int(v))
	}
	if maxUsedEvent >= int(managerLen) {
		log.Logger.Error().Msgf("Event enum value %d exceeds manager array size %d",
			maxUsedEvent, managerLen)
		panic("too many netsvrProtocol.Event")
	}
	Manager = manager{}
}

// Start 启动redis队列
func Start() {
	redisClientMp := make(map[string]*redis.Client)
	makeRedisClient(redisClientMp, configs.Config.RedisQueue.OnOpen)
	makeRedisClient(redisClientMp, configs.Config.RedisQueue.OnMessage)
	makeRedisClient(redisClientMp, configs.Config.RedisQueue.OnClose)
	queueMp := make(map[string]*Queue)
	Manager[int(netsvrProtocol.Event_OnOpen)] = makeQueue(redisClientMp, queueMp, configs.Config.RedisQueue.OnOpen)
	Manager[int(netsvrProtocol.Event_OnMessage)] = makeQueue(redisClientMp, queueMp, configs.Config.RedisQueue.OnMessage)
	Manager[int(netsvrProtocol.Event_OnClose)] = makeQueue(redisClientMp, queueMp, configs.Config.RedisQueue.OnClose)
	for _, q := range queueMp {
		log.Logger.Info().Int("pid", os.Getpid()).Str("redisKey", q.redisKey).Str("keyType", q.keyType).Str("address", q.redisClient.Options().Addr).Msg("RedisQueue start")
	}
}

// Shutdown 停止redis队列
func Shutdown() {
	for _, q := range Manager {
		if q != nil {
			q.Close()
			log.Logger.Info().Int("pid", os.Getpid()).Str("redisKey", q.redisKey).Str("keyType", q.keyType).Str("address", q.redisClient.Options().Addr).Msg("RedisQueue shutdown")
		}
	}
}

// makeRedisClient 创建一个redis客户端
func makeRedisClient(redisClientMp map[string]*redis.Client, queueConfig configs.RedisQueue) {
	if queueConfig.Address == "" || queueConfig.Key == "" {
		// 没有配置
		return
	}
	redisId := fmt.Sprintf("Address%sDB%d", queueConfig.Address, queueConfig.DB)
	if redisClientMp[redisId] != nil {
		// 已经初始化过了
		return
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     queueConfig.Address,
		Password: queueConfig.Password,
		DB:       queueConfig.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
	defer cancel()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Logger.Error().Err(err).Msgf("redisQueue queue init failed")
		time.Sleep(time.Millisecond * 300)
		panic(fmt.Sprintf("redisQueue queue init failed %v", err))
	}
	redisClientMp[redisId] = redisClient
}

// makeQueue 创建一个队列
func makeQueue(redisClientMp map[string]*redis.Client, queueMp map[string]*Queue, queueConfig configs.RedisQueue) *Queue {
	if queueConfig.Address == "" || queueConfig.Key == "" {
		// 没有配置
		return nil
	}
	queueId := fmt.Sprintf("Address%sDB%dKey%sKeyType%s", queueConfig.Address, queueConfig.DB, queueConfig.Key, queueConfig.KeyType)
	if queueMp[queueId] != nil {
		// 已经初始化过了，说明同一个队列可以处理多个Event，直接返回
		return queueMp[queueId]
	}
	redisId := fmt.Sprintf("Address%sDB%d", queueConfig.Address, queueConfig.DB)
	if redisClientMp[redisId] == nil {
		panic(fmt.Sprintf("redisQueue queue init failed %s", redisId))
	}
	q := newQueue(redisClientMp[redisId], queueConfig, 256)
	queueMp[queueId] = q
	return q
}
