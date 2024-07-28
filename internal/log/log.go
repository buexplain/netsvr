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
	"github.com/lesismal/nbio/logging"
	"github.com/rs/zerolog"
	"netsvr/configs"
	"netsvr/pkg/log"
)

var Logger zerolog.Logger

func init() {
	Logger = log.New(configs.Config.GetLogLevel(), configs.Config.GetLogFile())
	logging.DefaultLogger = newLoggingSubstitute(&Logger)
}

type loggingSubstitute struct {
	zero *zerolog.Logger
}

func newLoggingSubstitute(zero *zerolog.Logger) logging.Logger {
	return &loggingSubstitute{
		zero: zero,
	}
}

func (r *loggingSubstitute) SetLevel(_ int) {
}

func (r *loggingSubstitute) Debug(_ string, _ ...interface{}) {
}

func (r *loggingSubstitute) Info(_ string, _ ...interface{}) {
}

func (r *loggingSubstitute) Warn(format string, v ...interface{}) {
	r.zero.Warn().CallerSkipFrame(2).Msgf(format, v...)
}

func (r *loggingSubstitute) Error(format string, v ...interface{}) {
	r.zero.Error().CallerSkipFrame(2).Msgf(format, v...)
}
