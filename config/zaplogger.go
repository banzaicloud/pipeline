package config

import (
	"sync"

	"github.com/banzaicloud/pipeline/internal/platform/zaplog"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var zaplogger *zap.Logger
var zaploggerOnce sync.Once

// ZapLogger is a configured zap logger.
func ZapLogger() *zap.Logger {
	zaploggerOnce.Do(func() { zaplogger = zapLogger() })

	return zaplogger
}

func zapLogger() *zap.Logger {
	logger := zaplog.NewLogger(zaplog.Config{
		Level:  viper.GetString("logging.loglevel"),
		Format: viper.GetString("logging.logformat"),
	})

	return logger
}
