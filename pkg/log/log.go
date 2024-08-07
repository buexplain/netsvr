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

// Package log 日志模块
package log

import (
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"runtime/debug"
	"time"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DisableSampling(true)
	zerolog.ErrorStackMarshaler = func(_ error) interface{} {
		return string(debug.Stack())
	}
}

func New(lvl zerolog.Level, filename string) zerolog.Logger {
	var logger zerolog.Logger
	if filename == "" {
		w := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339Nano}
		logger = zerolog.New(w).Level(lvl).With().Caller().Timestamp().Logger()
	} else {
		//非debug模式，打印到日志文件
		w := &lumberjack.Logger{
			MaxSize:    1024 * 1024 * 255,
			MaxBackups: 3,
			MaxAge:     30,
			Compress:   false,
			Filename:   filename,
		}
		logger = zerolog.New(w).Level(lvl).With().Timestamp().Logger()
	}
	return logger
}
