// Package log 日志模块
package log

import (
	"github.com/lesismal/nbio/logging"
	"github.com/rs/zerolog"
	"netsvr/configs"
	"netsvr/pkg/log"
	"netsvr/pkg/wd"
	"path/filepath"
)

var Logger zerolog.Logger

func init() {
	Logger = log.New(configs.Config.GetLogLevel(), filepath.Join(wd.RootPath, "log/netSvr.log"))
	logging.DefaultLogger = log.NewLoggingSubstitute(&Logger)
}
