package log

import (
	"glt-calendar-service/settings/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sync"
	"time"
)

var (
	loggerInstance *zap.Logger
	once           sync.Once
)

func GetLogger() *zap.Logger {
	once.Do(func() {
		loggerInstance = initLogger()
	})
	return loggerInstance
}

func initLogger() *zap.Logger {
	logLevel := env.GetConfig().LogConfig.Level
	var atomicLevel zap.AtomicLevel
	switch logLevel {
	case "debug":
		atomicLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		atomicLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		atomicLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// 使用自定義配置
	config := zap.Config{
		Level:       atomicLevel,
		Development: true,
		Encoding:    "console",
		OutputPaths: []string{"stdout"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:      "timestamp",
			LevelKey:     "level",
			CallerKey:    "caller",
			MessageKey:   "message",
			EncodeLevel:  customLevelEncoder,
			EncodeTime:   customTimeEncoder,
			EncodeCaller: customCallerEncoder,
		},
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

func customLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000")) // yyyy-mm-dd hh:mm:ss.SSS
}

func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(caller.String())
}
