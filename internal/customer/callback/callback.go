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
	"encoding/json"
	"io"
	"net/http"
	"netsvr/configs"
	"netsvr/internal/log"
	"time"
)

var httpClient http.Client

func init() {
	httpClient = http.Client{Timeout: configs.Config.Customer.CallbackApiDeadline}
}

type OnOpenReq struct {
	//websocket能够承载的数据类型，1：TextMessage，2：BinaryMessage
	MessageType   int8     `json:"messageType"`
	UniqId        string   `json:"uniqId"`
	RawQuery      string   `json:"rawQuery"`
	SubProtocols  []string `json:"subProtocols"`
	XForwardedFor string   `json:"xForwardedFor"`
	XRealIp       string   `json:"xRealIp"`
	RemoteAddr    string   `json:"remoteAddr"`
}

type OnOpenResp struct {
	// 是否允许连接
	Allow bool `json:"allow"`
	// 新的session，传递了丢弃现有的，赋予新的
	NewSession string `json:"newSession"`
	// 新的客户id，传递了丢弃现有的，赋予新的
	NewCustomerId string `json:"newCustomerId"`
	// 新的主题，传递了丢弃现有的，赋予新的
	NewTopics []string `json:"newTopics"`
	// 需要发给客户的数据，传递了则转发给客户，注意，如果是BinaryMessage，则需要base64编码
	Data string `json:"data"`
}

func OnOpen(req *OnOpenReq) *OnOpenResp {
	paramsBytes, err := json.Marshal(req)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Format the callback.OnOpenReq error")
		return nil
	}
	resp, err := httpClient.Post(configs.Config.Customer.OnOpenCallbackApi, "application/json", bytes.NewReader(paramsBytes))
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
	data := &OnOpenResp{}
	err = json.Unmarshal(body, data)
	if err != nil {
		log.Logger.Error().Err(err).Str("body", string(body)).Msg("Parse callback.OnOpenResp error")
		return nil
	}
	return data
}

type OnCloseReq struct {
	UniqId     string   `json:"uniqId"`
	CustomerId string   `json:"customerId"`
	Session    string   `json:"session"`
	Topics     []string `json:"topics"`
}

func OnClose(req *OnCloseReq) {
	paramsBytes, err := json.Marshal(req)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Format the callback.OnCloseReq error")
		return
	}
	client := http.Client{Timeout: time.Second * 10}
	resp, err := client.Post(configs.Config.Customer.OnCloseCallbackApi, "application/json", bytes.NewReader(paramsBytes))
	if err != nil {
		log.Logger.Error().Err(err).Msg("Send callback.OnCloseReq error")
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	_, _ = io.ReadAll(resp.Body)
}
