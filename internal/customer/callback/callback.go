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

// Package callback 回调脚本
package callback

import (
	"bytes"
	"github.com/buexplain/netsvr-protocol-go/v5/netsvrProtocol"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"netsvr/configs"
	"netsvr/internal/log"
)

var httpClient *http.Client

func init() {
	httpClient = &http.Client{Timeout: configs.Config.Customer.CallbackApiDeadline}
}

func OnOpen(req *netsvrProtocol.ConnOpen) *netsvrProtocol.ConnOpenResp {
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Format the callback.OnOpenReq error")
		return nil
	}
	httpReq, err := http.NewRequest(
		http.MethodPost,
		configs.Config.Customer.OnOpenCallbackApi,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Send callback.OnOpenReq error")
		return nil
	}
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("Accept", "application/x-protobuf")
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Send callback.OnOpenReq error")
		return nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Read callback.OnOpenResp error")
		return nil
	}
	data := &netsvrProtocol.ConnOpenResp{}
	err = proto.Unmarshal(body, data)
	if err != nil {
		log.Logger.Error().Err(err).Str("body", string(body)).Msg("Parse callback.OnOpenResp error")
		return nil
	}
	return data
}

func OnClose(req *netsvrProtocol.ConnClose) {
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Format the callback.OnCloseReq error")
		return
	}
	resp, err := httpClient.Post(
		configs.Config.Customer.OnCloseCallbackApi,
		"application/x-protobuf",
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Send callback.OnCloseReq error")
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == http.StatusOK {
		_, _ = io.ReadAll(resp.Body)
	}
}
