/**
* Copyright 2024 buexplain@qq.com
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

package log

import "log/slog"

// LoggerInterface 日志接口，默认实现是 log/slog 的默认 logger
type LoggerInterface interface {
	// Debug 写 debug 日志
	Debug(msg string, args ...any)
	// Info 写 info 日志
	Info(msg string, args ...any)
	// Warn 写 warn 日志
	Warn(msg string, args ...any)
	// Error 写 error 日志
	Error(msg string, args ...any)
}

var logger LoggerInterface

func init() {
	SetLogger(slog.Default())
}

// SetLogger 替换 SDK 使用的日志实现
func SetLogger(l LoggerInterface) {
	logger = l
}

// GetLogger 获取 SDK 当前使用的日志实现
func GetLogger() LoggerInterface {
	return logger
}

// Debug 写 debug 日志
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info 写 info 日志
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn 写 warn 日志
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error 写 error 日志
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}
