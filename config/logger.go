package config

import (
	"sync"

	"github.com/banzaicloud/logrus-runtime-formatter"
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
	logger := logrus.New()

	level, err := logrus.ParseLevel(viper.GetString("logging.loglevel"))
	if err != nil {
		level = logrus.InfoLevel
	}

	logger.Level = level

	var childFormatter logrus.Formatter

	switch viper.GetString("log.logformat") {
	case "json":
		childFormatter = new(logrus.JSONFormatter)

	default:
		textFormatter := new(logrus.TextFormatter)
		textFormatter.FullTimestamp = true
		childFormatter = textFormatter
	}

	logger.Formatter = &runtime.Formatter{ChildFormatter: childFormatter}

	return logger
}
