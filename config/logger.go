package config

import (
	"sync"

	"github.com/banzaicloud/logrus-runtime-formatter"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger *logrus.Logger
var loggerOnce sync.Once

// Logger is a configured Logrus logger
func Logger() *logrus.Logger {
	loggerOnce.Do(func() { logger = newLogger() })

	return logger
}

func newLogger() *logrus.Logger {
	logger := log.NewLogger(log.Config{
		Level:  viper.GetString("logging.loglevel"),
		Format: viper.GetString("logging.logformat"),
	})

	logger.Formatter = &runtime.Formatter{ChildFormatter: logger.Formatter}

	return logger
}
