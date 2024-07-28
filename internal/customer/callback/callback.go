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

// Package callback 回调脚本的反射
package callback

import (
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"log/slog"
	"netsvr/configs"
	"os"
)

var OnOpen func(uniqId string, messageType int8, rawQuery string, subProtocols []string, XForwardedFor string, xRealIp string, remoteAddr string) (sendToCustomerData []byte, closeConnection bool)
var OnClose func(uniqId string, customerId string, session string, topics []string)

func init() {
	// 没有配置回调脚本文件，则直接返回
	if configs.Config.Customer.CallbackScriptFile == "" {
		return
	}
	// 读取回调脚本文件
	script, err := os.ReadFile(configs.Config.Customer.CallbackScriptFile)
	if err != nil {
		slog.Error("Config Customer.CallbackScriptFile read failed", "error", err)
		os.Exit(1)
	}
	// 创建解释器
	interpreter := interp.New(interp.Options{})
	if err := interpreter.Use(stdlib.Symbols); err != nil {
		slog.Error("interpreter use stdlib.Symbols failed", "error", err)
		os.Exit(1)
	}
	if _, err := interpreter.Eval(string(script)); err != nil {
		slog.Error("interpreter eval Customer.CallbackScriptFile failed", "error", err)
		os.Exit(1)
	}
	// 反射回调脚本
	if onOpenReflect, err := interpreter.Eval("configs.OnOpen"); err != nil {
		slog.Error("interpreter reflect Customer.CallbackScriptFile OnOpen function failed", "error", err)
		os.Exit(1)
	} else {
		OnOpen = onOpenReflect.Interface().(func(uniqId string, messageType int8, rawQuery string, subProtocols []string, XForwardedFor string, xRealIp string, remoteAddr string) (sendToCustomerData []byte, closeConnection bool))
	}
	if onCloseReflect, err := interpreter.Eval("configs.OnClose"); err != nil {
		slog.Error("iinterpreter reflect Customer.CallbackScriptFile OnClose function failed", "error", err)
		os.Exit(1)
	} else {
		OnClose = onCloseReflect.Interface().(func(uniqId string, customerId string, session string, topics []string))
	}
}
