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

// Package callback 连接打开、发送消息、关闭的回调接口，会发送http的post调用，header头是application/x-protobuf
package callback

import (
	"bytes"
	"fmt"
	"github.com/buexplain/netsvr-protocol-go/v6/netsvrProtocol"
	"github.com/panjf2000/gnet/v2/pkg/pool/byteslice"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"netsvr/configs"
	"netsvr/internal/log"
	internalMetrics "netsvr/internal/metrics"
	"unsafe"
)

var httpClient *http.Client

func init() {
	if configs.Config.Callback.OnOpenApi != "" || configs.Config.Callback.OnMessageApi != "" || configs.Config.Callback.OnCloseApi != "" {
		httpClient = &http.Client{Timeout: configs.Config.Callback.Timeout}
	}
}

// marshalAppendPooled 将 message 序列化到 byteslice 池；返回的 cleanup 须在请求体被 http 客户端读完后再调用（例如 defer）。
func marshalAppendPooled(message proto.Message) (data []byte, cleanup func(), err error) {
	opts := proto.MarshalOptions{}
	size := opts.Size(message)
	if size > 0 {
		pooled := byteslice.Get(size)
		data, err = opts.MarshalAppend(pooled[:0], message)
		if err != nil {
			byteslice.Put(pooled)
			return nil, nil, err
		}
		if unsafe.SliceData(data) != unsafe.SliceData(pooled) {
			byteslice.Put(pooled)
			return data, func() {}, nil
		}
		return data, func() { byteslice.Put(pooled) }, nil
	}
	data, err = opts.MarshalAppend(nil, message)
	if err != nil {
		return nil, nil, err
	}
	return data, func() {}, nil
}

// OnOpen websocket连接打开的回调接口
func OnOpen(req *netsvrProtocol.ConnOpen) (*netsvrProtocol.ConnOpenResp, error) {
	reqBytes, cleanup, err := marshalAppendPooled(req)
	if err != nil {
		internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessFailedCount].Meter.Mark(1)
		log.Logger.Error().Err(err).Msg("Format the netsvrProtocol.ConnOpen failed")
		return nil, err
	}
	defer cleanup()
	httpReq, err := http.NewRequest(
		http.MethodPost,
		configs.Config.Callback.OnOpenApi,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessFailedCount].Meter.Mark(1)
		log.Logger.Error().Err(err).Msgf("Send netsvrProtocol.ConnOpen to %s failed", configs.Config.Callback.OnOpenApi)
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("Accept", "application/x-protobuf")
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessFailedCount].Meter.Mark(1)
		log.Logger.Error().Err(err).Msgf("Send netsvrProtocol.ConnOpen to %s failed", configs.Config.Callback.OnOpenApi)
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	//写入成功：统计指标
	internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessSucceedCount].Meter.Mark(1)
	internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessSucceedByte].Meter.Mark(int64(len(reqBytes)))
	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		// 即使状态码不是200，也需要读取响应体以确保连接复用
		_, _ = io.Copy(io.Discard, resp.Body)
		err = fmt.Errorf("receive error http status code: %d", resp.StatusCode)
		log.Logger.Error().Err(err).Msgf("Read netsvrProtocol.ConnOpen from %s failed", configs.Config.Callback.OnOpenApi)
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Read netsvrProtocol.ConnOpen from %s failed", configs.Config.Callback.OnOpenApi)
		return nil, err
	}
	data := &netsvrProtocol.ConnOpenResp{}
	err = proto.Unmarshal(body, data)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Parse netsvrProtocol.ConnOpen failed")
		return nil, err
	}
	return data, nil
}

// OnMessage websocket连接发送消息的回调接口
func OnMessage(req *netsvrProtocol.Transfer) {
	reqBytes, cleanup, err := marshalAppendPooled(req)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Format the netsvrProtocol.Transfer failed")
		internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessFailedCount].Meter.Mark(1)
		return
	}
	defer cleanup()
	resp, err := httpClient.Post(
		configs.Config.Callback.OnMessageApi,
		"application/x-protobuf",
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessFailedCount].Meter.Mark(1)
		log.Logger.Error().Err(err).Msgf("Send netsvrProtocol.Transfer to %s failed", configs.Config.Callback.OnMessageApi)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	//写入成功：统计指标
	internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessSucceedCount].Meter.Mark(1)
	internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessSucceedByte].Meter.Mark(int64(len(reqBytes)))
	_, _ = io.Copy(io.Discard, resp.Body)
}

// OnClose websocket连接关闭的回调接口
func OnClose(req *netsvrProtocol.ConnClose) {
	reqBytes, cleanup, err := marshalAppendPooled(req)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Format the netsvrProtocol.ConnClose failed")
		internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessFailedCount].Meter.Mark(1)
		return
	}
	defer cleanup()
	resp, err := httpClient.Post(
		configs.Config.Callback.OnCloseApi,
		"application/x-protobuf",
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Send netsvrProtocol.ConnClose to %s failed", configs.Config.Callback.OnCloseApi)
		internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessFailedCount].Meter.Mark(1)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	//写入成功：统计指标
	internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessSucceedCount].Meter.Mark(1)
	internalMetrics.Registry[internalMetrics.ItemCallbackToBusinessSucceedByte].Meter.Mark(int64(len(reqBytes)))
	_, _ = io.Copy(io.Discard, resp.Body)
}
