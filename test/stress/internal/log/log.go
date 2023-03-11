// Package log 日志模块
package log

import (
	"github.com/lesismal/nbio/logging"
	"github.com/rs/zerolog"
	"netsvr/pkg/log"
	"netsvr/test/stress/configs"
)

var Logger zerolog.Logger

func init() {
	Logger = log.New(configs.Config.GetLogLevel(), "")
	logging.DefaultLogger = log.NewLoggingSubstitute(&Logger)
}
