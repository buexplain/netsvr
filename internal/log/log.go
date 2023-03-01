package log

import (
	"github.com/lesismal/nbio/logging"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
	"netsvr/configs"
	"os"
	"time"
)

var Logger zerolog.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DisableSampling(true)
	file := &lumberjack.Logger{
		MaxSize:    1024 * 1024 * 256,
		MaxBackups: 6,
		MaxAge:     1,
		Compress:   false,
		Filename:   configs.RootPath + "log/netSvr.log",
	}
	cls := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339Nano}
	multi := zerolog.MultiLevelWriter(file, cls)
	Logger = zerolog.New(multi).Level(configs.Config.GetLogLevel())
	logging.DefaultLogger = &loggingSubstitute{z: Logger}
}

type loggingSubstitute struct {
	z zerolog.Logger
}

func (r *loggingSubstitute) SetLevel(_ int) {
}

func (r *loggingSubstitute) Debug(format string, v ...interface{}) {
	r.z.Debug().Msgf(format, v...)
}

func (r *loggingSubstitute) Info(format string, v ...interface{}) {
	r.z.Info().Msgf(format, v...)
}

func (r *loggingSubstitute) Warn(format string, v ...interface{}) {
	r.z.Warn().Msgf(format, v...)
}

func (r *loggingSubstitute) Error(format string, v ...interface{}) {
	r.z.Error().Msgf(format, v...)
}
