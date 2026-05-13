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

package customer

import (
	"bytes"
	"github.com/gobwas/ws"
	"netsvr/internal/timer"
	"netsvr/internal/wsServer"
	"time"
)

// WriteMessage 发送数据
func WriteMessage(conn *wsServer.Conn, messageType ws.OpCode, data []byte) bool {
	fr := FrameObjPool.Get(messageType, data)
	defer FrameObjPool.Put(fr)
	return fr.WriteTo(conn)
}

// WriteClose 构建关闭帧，并发送关闭帧
func WriteClose(conn *wsServer.Conn, statusCode ws.StatusCode, err error) {
	WriteCloseFrame(conn, BuildCloseFrame(statusCode, err))
}

// WriteCloseFrame 发送关闭帧
func WriteCloseFrame(conn *wsServer.Conn, closeFrame []byte) {
	if conn.SetClosedFlagOnSafe() == false {
		//已经关闭，则直接返回
		return
	}
	err := conn.AsyncWriteOnSafe(closeFrame)
	if err == nil {
		//2秒后强制关闭连接，2秒足以覆盖绝大多数网络波动，足够让TCP层完成正常的状态流转，避免暴力切断
		timer.Timer.AfterFunc(time.Second*2, func() {
			conn.CloseOnSafe()
		})
	} else {
		//发送失败，则强制关闭连接
		conn.CloseOnSafe()
	}
}

// BuildCloseFrame 构建关闭帧
func BuildCloseFrame(statusCode ws.StatusCode, err error) []byte {
	buff := bytes.NewBuffer(make([]byte, 0, 127))
	payload := ws.NewCloseFrameBody(
		statusCode,
		err.Error(),
	)
	frame := ws.NewFrame(ws.OpClose, true, payload)
	_ = ws.WriteFrame(buff, frame)
	return buff.Bytes()
}

func CountCustomerIds(connList []*wsServer.Conn) int {
	customerIdSet := make(map[string]struct{}, len(connList))
	for _, conn := range connList {
		customerId := conn.GetCustomerIdOnSafe()
		if customerId != "" {
			customerIdSet[customerId] = struct{}{}
		}
	}
	return len(customerIdSet)
}

func GetCustomerIds(connList []*wsServer.Conn) (customerIds []string) {
	customerIdSet := make(map[string]struct{}, len(connList))
	for _, conn := range connList {
		customerId := conn.GetCustomerIdOnSafe()
		if customerId != "" {
			customerIdSet[customerId] = struct{}{}
		}
	}
	// 转换为 slice
	if len(customerIdSet) == 0 {
		return nil
	}
	customerIds = make([]string, 0, len(customerIdSet))
	for id := range customerIdSet {
		customerIds = append(customerIds, id)
	}
	return customerIds
}

func GetUniqIds(connList []*wsServer.Conn) (uniqIds []string) {
	uniqIds = make([]string, 0, len(connList))
	for _, conn := range connList {
		uniqIds = append(uniqIds, conn.GetUniqIdOnSafe())
	}
	return uniqIds
}
