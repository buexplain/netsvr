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

package cmd

import (
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/log"
	"netsvr/internal/objPool"
	workerManager "netsvr/internal/worker/manager"
)

// TopicPublishBulk 批量发布
func TopicPublishBulk(param []byte, _ *workerManager.ConnProcessor) {
	payload := objPool.TopicPublishBulk.Get()
	defer objPool.TopicPublishBulk.Put(payload)
	if err := proto.Unmarshal(param, payload); err != nil {
		log.Logger.Error().Err(err).Msg("Proto unmarshal netsvrProtocol.TopicPublishBulk failed")
		return
	}
	//当业务进程传递的topics的topic数量只有一个，data的datum数量是一个以上时，网关必须将所有的datum都发送给这个topic
	if len(payload.Topics) == 1 && len(payload.Data) > 1 {
		//先根据主题，获得主题下的所有uniqId
		uniqIds := topic.Topic.GetUniqIds(payload.Topics[0], objPool.UniqIdSlice)
		if uniqIds == nil {
			return
		}
		defer objPool.UniqIdSlice.Put(uniqIds)
		//再迭代所有uniqId
		uniqIdsAlias := *uniqIds //搞个别名，避免循环中解指针，提高性能
		//因为每个连接都要写入多条数据，所以直接协程并发发送
		coroutineNum := len(uniqIdsAlias)/50 + 1
		connCh := make(chan *websocket.Conn, coroutineNum)
		for i := 0; i < coroutineNum; i++ {
			go func(data [][]byte, connCh chan *websocket.Conn) {
				defer func() {
					_ = recover()
				}()
				var conn *websocket.Conn
				var index int
				for conn = range connCh {
					//将所有数据写入当前迭代到的连接
					for index = range data {
						if len(data[index]) == 0 {
							continue
						}
						if !customer.WriteMessage(conn, data[index]) {
							//写入失败，不再写入剩余的数据，而是跳出当前写数据的for循环，处理下一个conn
							break
						}
					}
				}
			}(payload.Data, connCh)
		}
		var uniqId string
		for _, uniqId = range uniqIdsAlias {
			conn := manager.Manager.Get(uniqId)
			if conn == nil {
				continue
			}
			connCh <- conn
		}
		close(connCh)
		return
	}
	//当业务进程传递的topics的topic数量与data的datum数量一致时，网关必须将同一下标的datum，发送给同一下标的topic
	if len(payload.Topics) > 0 && len(payload.Topics) == len(payload.Data) {
		var index int
		var currentTopic string
		var uniqId string
		//迭代所有的主题
		for index, currentTopic = range payload.Topics {
			//判断当前迭代的主题对应的数据是否有效
			if len(payload.Data[index]) == 0 {
				continue
			}
			//获得当前迭代的主题下的所有uniqId
			uniqIds := topic.Topic.GetUniqIds(currentTopic, objPool.UniqIdSlice)
			if uniqIds == nil {
				continue
			}
			//迭代所有uniqId
			uniqIdsAlias := *uniqIds //搞个别名，避免循环中解指针，提高性能
			//小于100个连接且topic数量最多两个，直接发送，这个条件的意思是：最多循环发送200个连接，超出限制都走协程并发发送
			if len(uniqIdsAlias) < 101 && len(payload.Topics) < 3 {
				for _, uniqId = range uniqIdsAlias {
					//根据uniqId获得对应的连接
					conn := manager.Manager.Get(uniqId)
					if conn == nil {
						continue
					}
					//将当前迭代的主题对应的数据写入到该连接
					customer.WriteMessage(conn, payload.Data[index])
				}
				//将uniqIds归还给内存池
				objPool.UniqIdSlice.Put(uniqIds)
				//跳过，处理下一个topic
				continue
			}
			//大于100个连接，或者是topic数量大于2，开启多协程发送
			coroutineNum := len(uniqIdsAlias)/100 + 1
			connCh := make(chan *websocket.Conn, coroutineNum)
			for i := 0; i < coroutineNum; i++ {
				go func(data []byte, connCh chan *websocket.Conn) {
					defer func() {
						_ = recover()
					}()
					for conn := range connCh {
						customer.WriteMessage(conn, data)
					}
				}(payload.Data[index], connCh)
			}
			for _, uniqId = range uniqIdsAlias {
				conn := manager.Manager.Get(uniqId)
				if conn == nil {
					continue
				}
				connCh <- conn
			}
			close(connCh)
			//将uniqIds归还给内存池
			objPool.UniqIdSlice.Put(uniqIds)
		}
	}
}
