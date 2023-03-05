// Package log 日志模块
package log

import (
	"github.com/lesismal/nbio/logging"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
	"netsvr/configs"
	"os"
	"runtime/debug"
	"time"
)

var Logger zerolog.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DisableSampling(true)
	zerolog.ErrorStackMarshaler = func(_ error) interface{} {
		return string(debug.Stack())
	}
	if configs.Config.GetLogLevel() == zerolog.DebugLevel {
		//debug模式直接打印到控制台
		w := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339Nano}
		Logger = zerolog.New(w).Level(configs.Config.GetLogLevel()).With().Caller().Timestamp().Logger()
	} else {
		//非debug模式，打印到日志文件
		w := &lumberjack.Logger{
			MaxSize:    1024 * 1024 * 256,
			MaxBackups: 6,
			MaxAge:     1,
			Compress:   false,
			Filename:   configs.RootPath + "log/netSvr.log",
		}
		Logger = zerolog.New(w).Level(configs.Config.GetLogLevel()).With().Timestamp().Logger()
	}
	logging.DefaultLogger = &loggingSubstitute{z: Logger}
}

type loggingSubstitute struct {
	z zerolog.Logger
}

func (r *loggingSubstitute) SetLevel(_ int) {
}

func (r *loggingSubstitute) Debug(format string, v ...interface{}) {
	r.z.Debug().CallerSkipFrame(2).Msgf(format, v...)
}

func (r *loggingSubstitute) Info(format string, v ...interface{}) {
	r.z.Info().CallerSkipFrame(2).Msgf(format, v...)
}

func (r *loggingSubstitute) Warn(format string, v ...interface{}) {
	r.z.Warn().CallerSkipFrame(2).Msgf(format, v...)
}

func (r *loggingSubstitute) Error(format string, v ...interface{}) {
	r.z.Error().CallerSkipFrame(2).Msgf(format, v...)
}
